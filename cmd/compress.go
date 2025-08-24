package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"tinypng-cli/internal/api"
)

var saveTypes = []string{"local", "aws_s3", "gcs"}
var metadataTypes = []string{"copyright", "creation", "location"}
var convertTypes = []string{"avif", "webp", "jpeg", "png", "*"}
var resizeTypes = []string{"scale", "fit", "cover", "thumb"}

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
		recursive                 bool
		maxUploadParallelism      int
		extensions                []string
		output                    string
		resizeHeight, resizeWidth int
		convertBG                 string
	)

	saveTo := FlagsProperty[string]{Flag: "save-to", Options: saveTypes}
	cmd.Flags().StringVar(&saveTo.Value, saveTo.Flag, "local", `save to: `+strings.Join(saveTypes, ","))

	// metadata config
	metadata := FlagsProperty[string]{Flag: "metadata", Options: metadataTypes}
	cmd.Flags().StringSliceVar(&metadata.Values, metadata.Flag, []string{}, `you can request the following metadata to the compressed file: 
`+strings.Join(metadataTypes, ",")+". location is JPEG only")

	// transform config
	convertTo := FlagsProperty[string]{Flag: "convert-to", Options: convertTypes}
	cmd.Flags().StringVar(&convertTo.Value, convertTo.Flag, "", `convert to specific type: 
`+strings.Join(convertTypes, ",")+". convert is only support between above types.")
	cmd.Flags().StringVar(&convertBG, "convert-bg", "", "transform background color hex value or white or black")

	// resize config
	resize := FlagsProperty[string]{Flag: "resize-method", Options: resizeTypes}
	cmd.Flags().StringVar(&resize.Value, resize.Flag, "", `resize method:`+strings.Join(resizeTypes, ",")+`
you can get more information about resize from official docs before start: https://tinypng.com/developers/reference#resizing-images`)
	cmd.Flags().IntVar(&resizeWidth, "resize-width", 0, "resize width")
	cmd.Flags().IntVar(&resizeHeight, "resize-height", 0, "resize height")

	cmd.Flags().StringVar(&output, "output", "", `compressed file output path.compressed file will be created beside by original file if output path is not set.`)

	cmd.Flags().IntVar(&maxUploadParallelism, "max-upload", 4, `max upload parallelism, valid only directory upload.
be aware of your upload bandwidth.`)
	cmd.Flags().BoolVar(&recursive, "recursive", false, `recursively read files from directory, valid only directory upload`)
	cmd.Flags().StringSliceVar(&extensions, "extensions", []string{"png", "jpg", "jpeg", "webp"}, `file extension filter, valid only directory upload`)

	// register flag completion
	saveTo.RegisterCompletion(cmd)
	metadata.RegisterCompletion(cmd)
	resize.RegisterCompletion(cmd)
	convertTo.RegisterCompletion(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path := args[0]

		saveConfig := saveConfig{
			metadata:     metadata.Values,
			convertTo:    convertTo.Value,
			convertBG:    convertBG,
			resizeMethod: resize.Value,
			resizeWidth:  resizeWidth,
			resizeHeight: resizeHeight,
		}

		client := api.GetTinyPNGClient()
		if api.IsUrl(path) {
			r, err := client.CompressFromUrl(path)
			if err != nil {
				return err
			}
			r.OriginalFile = path
			err = saveConfig.saveToLocal(output, r)
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
							err = saveConfig.saveToLocal(output, r)
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
				err = saveConfig.saveToLocal(output, r)
				if err != nil {
					return err
				}
			}
		}

		return nil
	}

	return cmd
}

type saveConfig struct {
	metadata                  []string
	convertTo, convertBG      string
	resizeWidth, resizeHeight int
	resizeMethod              string
}

var compressedSuffix = "-compressed."

func (c *saveConfig) saveToLocal(savePath string, result *api.CompressResult) error {
	var fullPath string
	if api.IsUrl(result.OriginalFile) {
		path, err := url.Parse(result.OriginalFile)
		if err != nil {
			return err
		}
		filename := strings.TrimSuffix(filepath.Base(path.Path), filepath.Ext(path.Path)) + compressedSuffix + result.Input.Suffix()
		fullPath = filepath.Join(savePath, filename)

	} else {
		if savePath == "" {
			fullPath = strings.TrimSuffix(result.OriginalFile, filepath.Ext(result.OriginalFile)) + compressedSuffix + result.Input.Suffix()
		} else {
			filename := strings.TrimSuffix(filepath.Base(result.OriginalFile), filepath.Ext(result.OriginalFile)) + compressedSuffix + result.Input.Suffix()
			fullPath = filepath.Join(savePath, filename)
		}
	}

	log.Printf("save to new local file %s\n", fullPath)

	normalDownload := true

	if c.metadata != nil && len(c.metadata) > 0 {
		err := api.DownloadWithMetadata(result.DownloadUrl, fullPath, c.metadata)
		if err != nil {
			return err
		}
		normalDownload = false
	}

	if c.convertTo != "" {
		err := api.DownloadWithConvert(result.DownloadUrl, fullPath, c.convertTo, c.convertBG)
		if err != nil {
			return err
		}
		normalDownload = false
	}

	if c.resizeMethod != "" {
		err := api.DownloadWithResize(result.DownloadUrl, fullPath, c.resizeMethod, c.resizeWidth, c.resizeHeight)
		if err != nil {
			return err
		}
		normalDownload = false
	}

	if normalDownload {
		err := api.Download(result.DownloadUrl, fullPath)
		if err != nil {
			return err
		}
	}

	return nil
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
