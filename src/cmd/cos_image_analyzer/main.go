// cos_Image_Analyzer finds all the meaningful differences of two COS Images
// (binary, package, commit, and release notes differences)
// Input:
//   (*ImageInfo) image1 - A struct that will store relevent info for image1
//   (*ImageInfo) image2 - A struct that will store relevent info for image2
//   (*FlagInfo) flagInfo - A struct that holds input preference from the user
// Output:
//   Based on "-output" flag, either "terminal" stdout (default) or "json" obj
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/binary"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/input"
	"cos.googlesource.com/cos/tools/src/cmd/cos_image_analyzer/internal/output"
)

func cosImageAnalyzer(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	imageDiff := &output.ImageDiff{}

	err := *new(error)
	if err := binary.GetBinaryInfo(image1, flagInfo); err != nil {
		return fmt.Errorf("failed to get GetBinaryInfo from image %v: %v", flagInfo.Image1, err)
	}
	if err := binary.GetBinaryInfo(image2, flagInfo); err != nil {
		return fmt.Errorf("failed to GetBinaryInfo from image %v: %v", flagInfo.Image2, err)
	}
	if err := image1.Rename(flagInfo); err != nil {
		return fmt.Errorf("failed to rename image %v: %v", flagInfo.Image1, err)
	}
	if err := image2.Rename(flagInfo); err != nil {
		return fmt.Errorf("failed to rename image %v: %v", flagInfo.Image2, err)
	}

	binaryDiff, err := binary.Diff(image1, image2, flagInfo)
	if err != nil {
		return fmt.Errorf("failed to get Binary Difference: %v", err)
	}
	imageDiff.BinaryDiff = binaryDiff

	output, err := imageDiff.Formater(flagInfo)
	if err != nil {
		return fmt.Errorf("failed to format image difference: %v", err)
	}
	if flagInfo.OutputSelected == "terminal" {
		imageDiff.Print(output)
	} else {
		fmt.Print(output)
	}
	return nil
}

// CallCosImageAnalyzer is wrapper that gets the images, calls cosImageAnalyzer, and cleans up
func CallCosImageAnalyzer(image1, image2 *input.ImageInfo, flagInfo *input.FlagInfo) error {
	if err := image1.MountImage(flagInfo.BinaryTypesSelected); err != nil {
		return fmt.Errorf("failed to mount first image %v: %v", flagInfo.Image1, err)
	}
	if err := image2.MountImage(flagInfo.BinaryTypesSelected); err != nil {
		return fmt.Errorf("failed to mount second image %v: %v", flagInfo.Image2, err)
	}
	if err := cosImageAnalyzer(image1, image2, flagInfo); err != nil {
		return fmt.Errorf("failed to call cosImageAnalyzer: %v", err)
	}
	return nil
}

func main() {
	if runtime.GOOS != "linux" {
		fmt.Printf("Error: This is a Linux tool, can not run on %s", runtime.GOOS)
	}

	flagInfo, err := input.ParseFlags()
	if err != nil {
		log.Printf("failed to parse flags: %v\n", err)
		os.Exit(1)
	}

	image1, image2, err := input.GetImages(flagInfo)
	if err != nil {
		log.Printf("failed to get images: %v", err)
		if err := image1.Cleanup(); err != nil {
			log.Printf("failed to clean up image %v: %v", flagInfo.Image1, err)
		}
		if err := image2.Cleanup(); err != nil {
			log.Printf("failed to clean up image %v: %v", flagInfo.Image2, err)
		}
		os.Exit(1)
	}

	if err = CallCosImageAnalyzer(image1, image2, flagInfo); err != nil {
		log.Printf("%v\n", err)
		if err := image1.Cleanup(); err != nil {
			log.Printf("failed to clean up image %v: %v", flagInfo.Image1, err)
		}
		if err := image2.Cleanup(); err != nil {
			log.Printf("failed to clean up image %v: %v", flagInfo.Image2, err)
		}
		os.Exit(1)
	}
	if err := image1.Cleanup(); err != nil {
		log.Printf("failed to clean up image %v: %v", flagInfo.Image1, err)
	}
	if err := image2.Cleanup(); err != nil {
		log.Printf("failed to clean up image %v: %v", flagInfo.Image2, err)
	}
}
