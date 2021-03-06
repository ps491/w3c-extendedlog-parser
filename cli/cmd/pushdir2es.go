package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
)

var pushdir2esCmd = &cobra.Command{
	Use:   "pushdir2es",
	Short: "Parse all files in some directory and push events to elasticsearch",
	Run: func(cmd *cobra.Command, args []string) {
		if batchsize <= 0 {
			batchsize = 5000
		}
		if len(input) == 0 {
			fatal(errors.New("specify an input directory"))
		}
		curdir, err := os.Getwd()
		fatal(err)
		curdir, err = filepath.Abs(curdir)
		fatal(err)
		input, err = filepath.Abs(input)
		fatal(err)

		inputFiles, err := findFiles(input, extension)
		fatal(err)

		if len(inputFiles) == 0 {
			fmt.Fprintln(os.Stderr, "No file to process.")
			return
		}
		fmt.Fprintln(os.Stderr, "Will process the following files")
		for _, fname := range inputFiles {
			fmt.Fprintf(os.Stderr, "- %s\n", fname)
		}
		fmt.Fprintln(os.Stderr)

		logger := log15.New()
		logger.SetHandler(log15.LvlFilterHandler(log15.LvlInfo, log15.StderrHandler))
		params := esParams{url: esURL, username: username, password: password}

		_, err = getESClient(params, logger)
		fatal(err)

		excludes := make(map[string]bool)
		for _, fName := range excludedFields {
			excludes[strings.ToLower(fName)] = true
		}
		excludes["date"] = true
		excludes["time"] = true

		for report := range uploadFilesES(params, inputFiles, batchsize, excludes, time.Month(onlyMonth), int(parallel), logger) {
			if report.err != nil {
				fmt.Fprintf(os.Stderr, "Failed to upload '%s': %s\n", report.filename, report.err.Error())
			} else {
				fmt.Fprintf(os.Stderr, "Uploaded '%s': %d lines\n", report.filename, report.nbLines)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(pushdir2esCmd)
	pushdir2esCmd.Flags().StringVar(&input, "input", "", "input directory")
	pushdir2esCmd.Flags().StringVar(&extension, "ext", "log", "only select input files with that extension")
	pushdir2esCmd.Flags().StringVar(&esURL, "url", "http://127.0.0.1:9200", "Elasticsearch connection URL")
	pushdir2esCmd.Flags().StringVar(&indexName, "index", "accesslogs", "Name of ES index to use")
	pushdir2esCmd.Flags().StringVar(&username, "username", "", "username for HTTP Basic Auth")
	pushdir2esCmd.Flags().StringVar(&password, "password", "", "password for HTTP Basic Auth")
	pushdir2esCmd.Flags().IntVar(&batchsize, "batchsize", 5000, "batch size to upload to ES")
	pushdir2esCmd.Flags().StringArrayVar(&excludedFields, "exclude", []string{}, "exclude that field from collection (can be repeated)")
	pushdir2esCmd.Flags().IntVar(&onlyMonth, "month", 0, "Only upload logs from that month")
	pushdir2esCmd.Flags().Uint8Var(&parallel, "parallel", 1, "number of parallel injectors")
}
