package images

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/AndroidPoet/playconsole-cli/internal/api"
	"github.com/AndroidPoet/playconsole-cli/internal/cli"
	"github.com/AndroidPoet/playconsole-cli/internal/output"
)

var ImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage screenshots and graphics",
	Long: `Manage screenshots, icons, and promotional graphics for your app.

Image types:
  - phoneScreenshots: Phone screenshots (up to 8)
  - sevenInchScreenshots: 7-inch tablet screenshots
  - tenInchScreenshots: 10-inch tablet screenshots
  - tvScreenshots: TV screenshots
  - wearScreenshots: Wear OS screenshots
  - featureGraphic: Feature graphic (1024x500)
  - icon: App icon (512x512)
  - tvBanner: TV banner (1280x720)
  - promoGraphic: Promo graphic (180x120)`,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List images for a locale and type",
	RunE:  runList,
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an image",
	RunE:  runUpload,
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an image",
	RunE:  runDelete,
}

var deleteAllCmd = &cobra.Command{
	Use:   "delete-all",
	Short: "Delete all images of a type",
	RunE:  runDeleteAll,
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync images from a directory",
	Long: `Sync images from a directory structure.

By default, sync appends uploaded images. Use --replace to delete
existing remote images for each discovered locale/type before uploading
the local files for that same locale/type.

Expected structure:
  screenshots/
    en-US/
      phoneScreenshots/
        1.png
        2.png
      featureGraphic/
        feature.png`,
	RunE: runSync,
}

var (
	locale          string
	imageType       string
	imageID         string
	filePath        string
	syncDir         string
	replaceExisting bool
	showProgress    bool
)

// Valid image types
var validImageTypes = []string{
	"phoneScreenshots",
	"sevenInchScreenshots",
	"tenInchScreenshots",
	"tvScreenshots",
	"wearScreenshots",
	"featureGraphic",
	"icon",
	"tvBanner",
	"promoGraphic",
}

func init() {
	// List flags
	listCmd.Flags().StringVar(&locale, "locale", "", "locale code (e.g., en-US)")
	listCmd.Flags().StringVar(&imageType, "type", "", "image type")
	listCmd.MarkFlagRequired("locale")
	listCmd.MarkFlagRequired("type")

	// Upload flags
	uploadCmd.Flags().StringVar(&locale, "locale", "", "locale code")
	uploadCmd.Flags().StringVar(&imageType, "type", "", "image type")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "path to image file")
	uploadCmd.Flags().BoolVar(&showProgress, "progress", false, "show upload progress")
	cli.AddStageFlag(uploadCmd)
	uploadCmd.MarkFlagRequired("locale")
	uploadCmd.MarkFlagRequired("type")
	uploadCmd.MarkFlagRequired("file")

	// Delete flags
	deleteCmd.Flags().StringVar(&locale, "locale", "", "locale code")
	deleteCmd.Flags().StringVar(&imageType, "type", "", "image type")
	deleteCmd.Flags().StringVar(&imageID, "id", "", "image ID to delete")
	cli.AddStageFlag(deleteCmd)
	deleteCmd.Flags().Bool("confirm", false, "confirm deletion")
	deleteCmd.MarkFlagRequired("locale")
	deleteCmd.MarkFlagRequired("type")
	deleteCmd.MarkFlagRequired("id")

	// Delete all flags
	deleteAllCmd.Flags().StringVar(&locale, "locale", "", "locale code")
	deleteAllCmd.Flags().StringVar(&imageType, "type", "", "image type")
	cli.AddStageFlag(deleteAllCmd)
	deleteAllCmd.Flags().Bool("confirm", false, "confirm deletion")
	deleteAllCmd.MarkFlagRequired("locale")
	deleteAllCmd.MarkFlagRequired("type")

	// Sync flags
	syncCmd.Flags().StringVar(&syncDir, "dir", "", "directory containing images")
	syncCmd.Flags().BoolVar(&showProgress, "progress", false, "show upload progress for each file")
	cli.AddStageFlag(syncCmd)
	syncCmd.Flags().BoolVar(&replaceExisting, "replace", false, "replace existing remote images for each synced locale/type")
	syncCmd.MarkFlagRequired("dir")

	ImagesCmd.AddCommand(listCmd)
	ImagesCmd.AddCommand(uploadCmd)
	ImagesCmd.AddCommand(deleteCmd)
	ImagesCmd.AddCommand(deleteAllCmd)
	ImagesCmd.AddCommand(syncCmd)
}

// ImageInfo represents image information
type ImageInfo struct {
	ID     string `json:"id"`
	URL    string `json:"url,omitempty"`
	SHA1   string `json:"sha1,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type progressReader struct {
	reader     io.Reader
	total      int64
	current    int64
	lastBucket int64
	label      string
}

func newProgressReader(reader io.Reader, total int64, label string) *progressReader {
	return &progressReader{
		reader:     reader,
		total:      total,
		lastBucket: -1,
		label:      label,
	}
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.current += int64(n)
		r.reportProgress()
	}

	if err == io.EOF && r.total > 0 && r.current >= r.total {
		r.reportFinalProgress()
	}

	return n, err
}

func (r *progressReader) reportProgress() {
	if r.total <= 0 {
		return
	}

	bucket := (r.current * 10) / r.total
	if bucket > 10 {
		bucket = 10
	}
	if bucket == r.lastBucket && r.current < r.total {
		return
	}

	r.lastBucket = bucket
	output.PrintInfo("Upload progress: %s %d%% (%s/%s)", r.label, percentComplete(r.current, r.total), formatBytes(r.current), formatBytes(r.total))
}

func (r *progressReader) reportFinalProgress() {
	if r.lastBucket == 10 {
		return
	}
	r.lastBucket = 10
	output.PrintInfo("Upload progress: %s 100%% (%s/%s)", r.label, formatBytes(r.total), formatBytes(r.total))
}

func percentComplete(current, total int64) int64 {
	if total <= 0 {
		return 0
	}
	if current >= total {
		return 100
	}
	return (current * 100) / total
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	suffixes := []string{"KiB", "MiB", "GiB", "TiB"}
	for _, suffix := range suffixes {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}

	return fmt.Sprintf("%.1f PiB", value/unit)
}

func validateImageType(t string) error {
	for _, valid := range validImageTypes {
		if t == valid {
			return nil
		}
	}
	return fmt.Errorf("invalid image type '%s'. Valid types: %s", t, strings.Join(validImageTypes, ", "))
}

func runList(cmd *cobra.Command, args []string) error {
	if err := cli.RequirePackage(cmd); err != nil {
		return err
	}

	if err := validateImageType(imageType); err != nil {
		return err
	}

	client, err := api.NewClient(cli.GetPackageName(), 60*time.Second)
	if err != nil {
		return err
	}

	edit, err := client.CreateEdit()
	if err != nil {
		return err
	}
	defer edit.Close()
	defer edit.Delete()

	images, err := edit.Images().List(client.GetPackageName(), edit.ID(), locale, imageType).Context(edit.Context()).Do()
	if err != nil {
		return err
	}

	result := make([]ImageInfo, 0, len(images.Images))
	for _, img := range images.Images {
		result = append(result, ImageInfo{
			ID:     img.Id,
			URL:    img.Url,
			SHA1:   img.Sha1,
			SHA256: img.Sha256,
		})
	}

	if len(result) == 0 {
		output.PrintInfo("No images found for %s/%s", locale, imageType)
		return nil
	}

	return output.Print(result)
}

func runUpload(cmd *cobra.Command, args []string) error {
	if err := cli.RequirePackage(cmd); err != nil {
		return err
	}

	submission, err := cli.GetEditSubmission(cmd, true)
	if err != nil {
		return err
	}

	if err := validateImageType(imageType); err != nil {
		return err
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("file not found: %s", absPath)
	}

	if cli.IsDryRun() {
		output.PrintInfo("Dry run: would upload %s to %s/%s", filepath.Base(absPath), locale, imageType)
		return nil
	}

	client, err := api.NewClient(cli.GetPackageName(), 2*time.Minute)
	if err != nil {
		return err
	}

	edit, err := client.CreateEdit()
	if err != nil {
		return err
	}
	defer edit.Close()

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	output.PrintInfo("Uploading: %s (%d bytes)", filepath.Base(absPath), info.Size())

	reader := io.Reader(file)
	if showProgress {
		reader = newProgressReader(file, info.Size(), filepath.Base(absPath))
	}

	image, err := edit.Images().Upload(client.GetPackageName(), edit.ID(), locale, imageType).Media(reader).Context(edit.Context()).Do()
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	if err := cli.ApplyEditSubmission(edit, submission); err != nil {
		return err
	}

	output.PrintSuccess("Image uploaded: %s", image.Image.Id)
	return output.Print(ImageInfo{
		ID:     image.Image.Id,
		SHA1:   image.Image.Sha1,
		SHA256: image.Image.Sha256,
	})
}

func runDelete(cmd *cobra.Command, args []string) error {
	if err := cli.RequirePackage(cmd); err != nil {
		return err
	}

	submission, err := cli.GetEditSubmission(cmd, true)
	if err != nil {
		return err
	}

	if err := validateImageType(imageType); err != nil {
		return err
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		return fmt.Errorf("use --confirm to delete image")
	}

	if cli.IsDryRun() {
		output.PrintInfo("Dry run: would delete image %s from %s/%s", imageID, locale, imageType)
		return nil
	}

	client, err := api.NewClient(cli.GetPackageName(), 60*time.Second)
	if err != nil {
		return err
	}

	edit, err := client.CreateEdit()
	if err != nil {
		return err
	}
	defer edit.Close()

	err = edit.Images().Delete(client.GetPackageName(), edit.ID(), locale, imageType, imageID).Context(edit.Context()).Do()
	if err != nil {
		return err
	}

	if err := cli.ApplyEditSubmission(edit, submission); err != nil {
		return err
	}

	output.PrintSuccess("Image deleted: %s", imageID)
	return nil
}

func runDeleteAll(cmd *cobra.Command, args []string) error {
	if err := cli.RequirePackage(cmd); err != nil {
		return err
	}

	submission, err := cli.GetEditSubmission(cmd, true)
	if err != nil {
		return err
	}

	if err := validateImageType(imageType); err != nil {
		return err
	}

	confirm, _ := cmd.Flags().GetBool("confirm")
	if !confirm {
		return fmt.Errorf("use --confirm to delete all images of type '%s'", imageType)
	}

	if cli.IsDryRun() {
		output.PrintInfo("Dry run: would delete all images from %s/%s", locale, imageType)
		return nil
	}

	client, err := api.NewClient(cli.GetPackageName(), 60*time.Second)
	if err != nil {
		return err
	}

	edit, err := client.CreateEdit()
	if err != nil {
		return err
	}
	defer edit.Close()

	_, err = edit.Images().Deleteall(client.GetPackageName(), edit.ID(), locale, imageType).Context(edit.Context()).Do()
	if err != nil {
		return err
	}

	if err := cli.ApplyEditSubmission(edit, submission); err != nil {
		return err
	}

	output.PrintSuccess("All images deleted for %s/%s", locale, imageType)
	return nil
}

func runSync(cmd *cobra.Command, args []string) error {
	if err := cli.RequirePackage(cmd); err != nil {
		return err
	}

	submission, err := cli.GetEditSubmission(cmd, true)
	if err != nil {
		return err
	}

	absDir, err := filepath.Abs(syncDir)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("directory not found: %s", absDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absDir)
	}

	uploaded := 0
	mutated := false

	// Walk directory: locale/imageType/files
	locales, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}

	type syncBatch struct {
		locale    string
		imageType string
		files     []string
	}

	batches := make([]syncBatch, 0)

	for _, localeEntry := range locales {
		if !localeEntry.IsDir() {
			continue
		}

		localeName := localeEntry.Name()
		localeDir := filepath.Join(absDir, localeName)

		types, err := os.ReadDir(localeDir)
		if err != nil {
			continue
		}

		for _, typeEntry := range types {
			if !typeEntry.IsDir() {
				continue
			}

			typeName := typeEntry.Name()
			if err := validateImageType(typeName); err != nil {
				continue
			}

			typeDir := filepath.Join(localeDir, typeName)
			fileNames, err := collectValidImageFiles(typeDir)
			if err != nil {
				output.PrintWarning("Failed to collect images from %s: %v", typeDir, err)
				continue
			}

			if len(fileNames) == 0 && !replaceExisting {
				continue
			}

			batches = append(batches, syncBatch{
				locale:    localeName,
				imageType: typeName,
				files:     fileNames,
			})
		}
	}

	if cli.IsDryRun() {
		wouldUpload := 0
		for _, batch := range batches {
			if replaceExisting {
				output.PrintInfo("Dry run: would replace existing images for %s/%s", batch.locale, batch.imageType)
			}

			for _, fileName := range batch.files {
				output.PrintInfo("Dry run: would upload %s to %s/%s", fileName, batch.locale, batch.imageType)
				wouldUpload++
			}
		}

		output.PrintSuccess("Would upload %d image(s)", wouldUpload)
		return nil
	}

	if len(batches) == 0 {
		output.PrintSuccess("Uploaded %d image(s)", uploaded)
		return nil
	}

	client, err := api.NewClient(cli.GetPackageName(), 5*time.Minute)
	if err != nil {
		return err
	}

	edit, err := client.CreateEdit()
	if err != nil {
		return err
	}
	defer edit.Close()

	for _, batch := range batches {
		if replaceExisting {
			output.PrintInfo("Replacing existing images for %s/%s", batch.locale, batch.imageType)
			ctx, cancel := edit.RequestContext()
			_, err := edit.Images().Deleteall(client.GetPackageName(), edit.ID(), batch.locale, batch.imageType).Context(ctx).Do()
			cancel()
			if err != nil {
				output.PrintWarning("Failed to delete existing images for %s/%s: %v", batch.locale, batch.imageType, err)
				continue
			}
			mutated = true
		}

		for _, fileName := range batch.files {
			filePath := filepath.Join(absDir, batch.locale, batch.imageType, fileName)
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				output.PrintWarning("Failed to stat %s: %v", filePath, err)
				continue
			}

			file, err := os.Open(filePath)
			if err != nil {
				output.PrintWarning("Failed to open %s: %v", filePath, err)
				continue
			}

			reader := io.Reader(file)
			if showProgress {
				reader = newProgressReader(file, fileInfo.Size(), filepath.Join(batch.locale, batch.imageType, fileName))
			}

			ctx, cancel := edit.RequestContext()
			_, err = edit.Images().Upload(client.GetPackageName(), edit.ID(), batch.locale, batch.imageType).Media(reader).Context(ctx).Do()
			cancel()
			file.Close()

			if err != nil {
				output.PrintWarning("Failed to upload %s: %v", fileName, err)
				continue
			}

			output.PrintInfo("Uploaded: %s/%s/%s", batch.locale, batch.imageType, fileName)
			uploaded++
			mutated = true
		}
	}

	if mutated {
		if err := cli.ApplyEditSubmission(edit, submission); err != nil {
			return err
		}
	} else if err := edit.Delete(); err != nil {
		return err
	}

	output.PrintSuccess("Uploaded %d image(s)", uploaded)
	return nil
}

func collectValidImageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
			continue
		}

		files = append(files, entry.Name())
	}

	sort.Strings(files)
	return files, nil
}
