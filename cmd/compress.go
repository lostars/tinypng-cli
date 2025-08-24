package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tinypng-cli/internal/api"
)

var saveTypes = []string{"local", "aws_s3", "gcs"}
var metadataTypes = []string{"copyright", "creation", "location"}

func CompressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compress <path>",
		Short: "Compress images",
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
		// local path
		output string
	)

	saveTo := FlagsProperty[string]{Flag: "save-to", Options: saveTypes}
	cmd.Flags().StringVar(&saveTo.Value, saveTo.Flag, "local", `save to: `+strings.Join(saveTypes, ","))

	metadata := FlagsProperty[string]{Flag: "metadata", Options: metadataTypes}
	cmd.Flags().StringSliceVar(&metadata.Values, metadata.Flag, []string{}, `you can request the following metadata to the compressed file: 
`+strings.Join(metadataTypes, ",")+". location is JPEG only")

	cmd.Flags().StringVar(&output, "output", "", `compressed file output path.compressed file will be created beside by original file if output path is not set.`)

	cmd.Flags().IntVar(&maxUploadParallelism, "max-upload", 4, `max upload parallelism, valid only directory upload.
be aware of your upload bandwidth.`)
	cmd.Flags().BoolVar(&recursive, "recursive", false, `recursively read files from directory, valid only directory upload`)
	cmd.Flags().StringSliceVar(&extensions, "extensions", []string{"png", "jpg", "jpeg", "webp"}, `file extension filter, valid only directory upload`)

	// register flag completion
	saveTo.RegisterCompletion(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path := args[0]

		client := api.GetTinyPNGClient()
		if api.IsUrl(path) {
			r, err := client.CompressFromUrl(path)
			if err != nil {
				return err
			}
			r.OriginalFile = path
			err = r.SaveToLocal(output, metadata.Values)
			if err != nil {
				return err
			}

		} else {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
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
							r, err := client.CompressFromFile(file)
							if err != nil {
								fmt.Println(err)
							}
							r.OriginalFile = file
							err = r.SaveToLocal(output, metadata.Values)
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
				r, err := client.CompressFromFile(path)
				if err != nil {
					return err
				}
				r.OriginalFile = path
				err = r.SaveToLocal(output, metadata.Values)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return cmd
}

func listFiles(path string, recursive bool, extensions []string) (chan string, error) {
	files := make(chan string, 100)
	sendFile := func(path string) {
		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(path), ext) {
				files <- path
				break
			}
		}
	}

	if recursive {
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				sendFile(path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			sendFile(filepath.Join(path, entry.Name()))
		}
	}
	return files, nil
}
