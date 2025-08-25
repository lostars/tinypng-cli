package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tinypng-cli/internal/api"
)

func CompressWebCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "web-compress <path>",
		Short: "Compress images using web page api",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("must provide a file, directory or url")
			}
			return nil
		},
	}

	var (
		recursive            bool
		maxUploadParallelism int
		extensions           []string
		output               string
	)

	cmd.Flags().StringVar(&output, "output", "", `compressed file output path. compressed file will be created beside by original file if output path is not set.`)

	cmd.Flags().IntVar(&maxUploadParallelism, "max-upload", 4, `max upload parallelism, valid only directory upload.
be aware of your upload bandwidth.`)
	cmd.Flags().BoolVar(&recursive, "recursive", false, `recursively read files from directory, valid only directory upload`)
	cmd.Flags().StringSliceVar(&extensions, "extensions", []string{"png", "jpg", "jpeg", "webp"}, `file extension filter, valid only directory upload`)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path := args[0]

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		client := api.GetTinyPNGWebClient()
		if info.IsDir() {
			files, err := listFiles(path, recursive, extensions)
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			for i := 0; i < maxUploadParallelism; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					for file := range files {
						log.Printf("compressing file: %s\n", file)
						r, err := client.WebCompressFromFile(file)
						if err != nil {
							fmt.Println(err)
						}
						err = saveToLocal(output, file, r)
						if err != nil {
							fmt.Println(err)
						}
					}
				}(i)
			}

			close(files)
			wg.Wait()
			fmt.Println("compressing done.")

		} else {
			r, err := client.WebCompressFromFile(path)
			if err != nil {
				return err
			}
			err = saveToLocal(output, path, r)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return cmd
}

func saveToLocal(output string, originalFile string, result *api.WebDownloadResult) error {
	fullPath := ""
	if output == "" {
		fullPath = strings.TrimSuffix(originalFile, filepath.Ext(originalFile)) + compressedSuffix + api.SuffixFromMIME(result.Type)
	} else {
		filename := strings.TrimSuffix(filepath.Base(originalFile), filepath.Ext(originalFile)) + compressedSuffix + api.SuffixFromMIME(result.Type)
		fullPath = filepath.Join(output, filename)
	}

	// save file
	resp, err := http.Get(result.Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}
	err = api.WriteFileFromResp(resp, fullPath)
	if err != nil {
		return err
	}
	return nil
}
