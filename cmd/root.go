package cmd

import (
	//"crypto/tls"
	"net/url"
	"os/user"
	"os"
	"fmt"
	"errors"


	"github.com/helviojunior/enumdns/internal"
	"github.com/helviojunior/enumdns/internal/ascii"
	"github.com/helviojunior/enumdns/pkg/log"
	"github.com/helviojunior/enumdns/pkg/runner"
	"github.com/helviojunior/enumdns/pkg/readers"
    resolver "github.com/helviojunior/gopathresolver"
	"github.com/spf13/cobra"
)

var (
	opts = &runner.Options{}
	fileOptions = &readers.FileReaderOptions{}
	tProxy = ""
)

var rootCmd = &cobra.Command{
	Use:   "enumdns",
	Short: "enumdns is a modular DNS recon tool",
	Long:  ascii.Logo(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		
		usr, err := user.Current()
	    if err != nil {
	       return err
	    }

	    opts.Writer.UserPath = usr.HomeDir

		if opts.Logging.Silence {
			log.EnableSilence()
		}

		if opts.Logging.Debug && !opts.Logging.Silence {
			log.EnableDebug()
			log.Debug("debug logging enabled")
		}

        if opts.Writer.TextFile != "" {

        	opts.Writer.TextFile, err = resolver.ResolveFullPath(opts.Writer.TextFile)
	        if err != nil {
	            return err
	        }

            opts.Writer.Text = true
        }

        //Check Proxy config
        if tProxy != "" {
        	u, err := url.Parse(tProxy)
        	if err != nil {
	        	return errors.New("Error parsing URL: " + err.Error())
	        }

        	_, err = internal.FromURL(u, nil)
        	if err != nil {
	        	return errors.New("Error parsing URL: " + err.Error())
	        }
	        opts.Proxy = u
	        fileOptions.ProxyUri = opts.Proxy

			port := u.Port()
			if port == "" {
				port = "1080"
			}
	        log.Warn("Setting proxy to " + u.Scheme + "://" + u.Hostname() + ":" + port)
        }else{
        	opts.Proxy = nil
        }
        
		return nil
	},
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceErrors = true
	err := rootCmd.Execute()
	if err != nil {
		var cmd string
		c, _, cerr := rootCmd.Find(os.Args[1:])
		if cerr == nil {
			cmd = c.Name()
		}

		v := "\n"

		if cmd != "" {
			v += fmt.Sprintf("An error occured running the `%s` command\n", cmd)
		} else {
			v += "An error has occured. "
		}

		v += "The error was:\n\n" + fmt.Sprintf("```%s```", err)
		fmt.Println(ascii.Markdown(v))

		os.Exit(1)
	}
}

func init() {
	// Disable Certificate Validation (Globally)
	//http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Debug, "debug-log", "D", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&opts.Logging.Silence, "quiet", "q", false, "Silence (almost all) logging")

	rootCmd.PersistentFlags().StringVarP(&opts.Writer.TextFile, "write-text-file", "o", "", "The file to write Text lines to")
    

	//rootCmd.PersistentFlags().BoolVarP(&opts.DnsOverHttps.SkipSSLCheck, "ssl-insecure", "K", true, "SSL Insecure")
	rootCmd.PersistentFlags().StringVarP(&tProxy, "proxy", "X", "", "Proxy to pass traffic through: <scheme://ip:port> (e.g., http://user:pass@proxy_host:1080")
	//rootCmd.PersistentFlags().StringVarP(&opts.DnsOverHttps.ProxyUser, "proxy-user", "", "", "Proxy User")
	//rootCmd.PersistentFlags().StringVarP(&opts.DnsOverHttps.ProxyPassword, "proxy-pass", "", "", "Proxy Password")

}
