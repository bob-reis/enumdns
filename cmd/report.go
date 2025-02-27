package cmd

import (
	"bufio"
    "encoding/json"
    "fmt"
    "io"
    "os"

    "github.com/helviojunior/enumdns/internal/ascii"
    "github.com/helviojunior/enumdns/pkg/database"
    "github.com/helviojunior/enumdns/pkg/log"
    "github.com/helviojunior/enumdns/pkg/models"
    "github.com/helviojunior/enumdns/pkg/writers"
    "github.com/spf13/cobra"
    "gorm.io/gorm/clause"
)

var reportCmd = &cobra.Command{
    Use:   "report",
    Short: "Work with enumdns reports",
    Long: ascii.LogoHelp(ascii.Markdown(`
# report

Work with enumdns reports.
`)),
}

func init() {
    rootCmd.AddCommand(reportCmd)
}


func convertFromDbTo(from string, writer writers.Writer) error {
	log.Info("starting conversion...")

    var results = []*models.Result{}
    conn, err := database.Connection(fmt.Sprintf("sqlite:///%s", from), true, false)
    if err != nil {
        return err
    }

    if err := conn.Model(&models.Result{}).Preload(clause.Associations).Where("`exists` = ?", 1).Find(&results).Error; err != nil {
        return err
    }

    for _, result := range results {
        if err := writer.Write(result); err != nil {
            return err
        }
    }

    log.Info("converted from a database", "rows", len(results))
    return nil
}
func convertFromJsonlTo(from string, writer writers.Writer) error {
	log.Info("starting conversion...")

    file, err := os.Open(from)
    if err != nil {
        return err
    }
    defer file.Close()

    var c = 0

    reader := bufio.NewReader(file)
    for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            if err == io.EOF {
                if len(line) == 0 {
                    break // End of file
                }
                // Handle the last line without '\n'
            } else {
                return err
            }
        }

        var result models.Result
        if err := json.Unmarshal(line, &result); err != nil {
            log.Error("could not unmarshal JSON line", "err", err)
            continue
        }

        if err := writer.Write(&result); err != nil {
            return err
        }
        c++

        if err == io.EOF {
            break
        }
    }

    log.Info("converted from a JSON Lines file", "rows", c)
    return nil
}