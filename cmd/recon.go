package cmd

import (
    "errors"
    "log/slog"
    "os"
    "fmt"

    "github.com/helviojunior/enumdns/internal/ascii"
    "github.com/helviojunior/enumdns/internal/tools"
    "github.com/helviojunior/enumdns/pkg/log"
    "github.com/helviojunior/enumdns/pkg/runner"
    //"github.com/helviojunior/enumdns/pkg/database"
    "github.com/helviojunior/enumdns/pkg/writers"
    "github.com/helviojunior/enumdns/pkg/readers"
    "github.com/spf13/cobra"
)

var reconRunner *runner.Recon

var reconWriters = []writers.Writer{}
var reconCmd = &cobra.Command{
    Use:   "recon",
    Short: "Perform recon enumeration",
    Long: ascii.LogoHelp(ascii.Markdown(`
# recon

Perform recon enumeration.

By default, enumdns will only show information regarding the recon process. 
However, that is only half the fun! You can add multiple _writers_ that will 
collect information such as response codes, content, and more. You can specify 
multiple writers using the _--writer-*_ flags (see --help).
`)),
    Example: `
   - enumdns recon -d helviojunior.com.br -o enumdns.txt
   - enumdns recon -d helviojunior.com.br --write-jsonl
   - enumdns recon -L domains.txt --write-db`,
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        var err error

        // Annoying quirk, but because I'm overriding PersistentPreRun
        // here which overrides the parent it seems.
        // So we need to explicitly call the parent's one now.
        if err = rootCmd.PersistentPreRunE(cmd, args); err != nil {
            return err
        }

        // An slog-capable logger to use with drivers and runners
        logger := slog.New(log.Logger)

        // Configure writers that subcommand scanners will pass to
        // a runner instance.

        //The first one is the general writer (global user)
        w, err := writers.NewDbWriter("sqlite:///" + opts.Writer.UserPath +"/.enumdns.db", false)
        if err != nil {
            return err
        }
        reconWriters = append(reconWriters, w)

        //The second one is the STDOut
        if opts.Logging.Silence != true {
            w, err := writers.NewStdoutWriter()
            if err != nil {
                return err
            }
            w.WriteAll = true
            reconWriters = append(reconWriters, w)
        }
    
        if opts.Writer.Text {
            w, err := writers.NewTextWriter(opts.Writer.TextFile)
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if opts.Writer.Jsonl {
            w, err := writers.NewJsonWriter(opts.Writer.JsonlFile)
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if opts.Writer.Db {
            w, err := writers.NewDbWriter(opts.Writer.DbURI, opts.Writer.DbDebug)
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if opts.Writer.Csv {
            w, err := writers.NewCsvWriter(opts.Writer.CsvFile)
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if opts.Writer.ELastic {
            w, err := writers.NewElasticWriter(opts.Writer.ELasticURI)
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if opts.Writer.None {
            w, err := writers.NewNoneWriter()
            if err != nil {
                return err
            }
            reconWriters = append(reconWriters, w)
        }

        if len(reconWriters) == 0 {
            log.Warn("no writers have been configured. to persist probe results, add writers using --write-* flags")
        }

        // Get the runner up. Basically, all of the subcommands will use this.
        reconRunner, err = runner.NewRecon(logger, *opts, reconWriters)
        if err != nil {
            return err
        }

        fileOptions.DnsServer = opts.DnsServer + ":" + fmt.Sprintf("%d", opts.DnsPort)

        return nil
    },
    PreRunE: func(cmd *cobra.Command, args []string) error {
        if opts.DnsSuffix == "" && fileOptions.DnsSuffixFile == "" {
            return errors.New("a DNS suffix or DNS suffix file must be specified")
        }

        if fileOptions.DnsSuffixFile != "" {
            if !tools.FileExists(fileOptions.DnsSuffixFile) {
                return errors.New("DNS suffix file is not readable")
            }
        }

        return nil
    },
    Run: func(cmd *cobra.Command, args []string) {

        //Check DNS connectivity
        _, err := tools.GetValidDnsSuffix(fileOptions.DnsServer, "google.com.", opts.Proxy)
        if err != nil {
            log.Error("Error checking DNS connectivity", "err", err)
            os.Exit(2)
        }

        log.Debug("starting DNS recon")

        dnsSuffix := []string{}
        enumeratedDomains := []string{}
        reader := readers.NewFileReader(fileOptions)
        total := 0

        if fileOptions.DnsSuffixFile != "" {
            log.Debugf("Reading dns suffix file: %s", fileOptions.DnsSuffixFile)
            if err := reader.ReadDnsList(&dnsSuffix); err != nil {
                log.Error("error in reader.Read", "err", err)
                log.Warn("If you are facing error related to 'SOA not found for domain' you can ignore it with -I option")
                os.Exit(2)
            }
        }else{
            //Check if DNS exists
            s, err := tools.GetValidDnsSuffix(fileOptions.DnsServer, opts.DnsSuffix, opts.Proxy)
            if err != nil {
                log.Error("invalid dns suffix", "suffix", opts.DnsSuffix, "err", err)
                os.Exit(2)
            }
            dnsSuffix = append(dnsSuffix, s)
        }
        log.Debugf("Loaded %s DNS name(s)", tools.FormatInt(len(dnsSuffix)))

        total = len(dnsSuffix)

        if len(dnsSuffix) == 0 {
            log.Error("DNS suffix list is empty")
            os.Exit(2)
        }

        for len(dnsSuffix) > 0 {
            log.Warnf("Enumerating %s DNS hosts", tools.FormatInt(total))

            go func() {
                defer close(reconRunner.Targets)

                ascii.HideCursor()

                for _, s := range dnsSuffix {
                    reconRunner.Targets <- s
                    enumeratedDomains = append(enumeratedDomains, s)
                }

            }()

            reconRunner.Run(total)

            dnsSuffix = []string{}

            for _, d := range reconRunner.Domains {
                if !tools.SliceHasStr(enumeratedDomains, d) && !tools.SliceHasStr(dnsSuffix, d) {
                    dnsSuffix = append(dnsSuffix, d)
                }
            }

            total = len(dnsSuffix)
            if total > 0 {
                log.Infof("%s new domain(s) found", tools.FormatInt(total))
                reconRunner.Reset()
            }

        }

        reconRunner.Close()

    },
}

func init() {
    rootCmd.AddCommand(reconCmd)
    
    reconCmd.Flags().StringVarP(&opts.DnsSuffix, "dns-name", "d", "", "Single DNS suffix. (ex: helviojunior.com.br)")
    reconCmd.Flags().StringVarP(&fileOptions.DnsSuffixFile, "dns-list", "L", "", "File containing a list of DNS names")
    
    reconCmd.Flags().BoolVarP(&fileOptions.IgnoreNonexistent, "IgnoreNonexistent", "I", false, "Ignore Nonexistent DNS suffix. Used only with --dns-list option.")
    
}