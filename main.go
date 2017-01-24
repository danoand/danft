package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	err               error
	tmp, fpath, fbase string
	args              []string

	apiUpEndPt = "https://danoutils.danocloud.com/uploadafile"
	apiDnEndPt = "https://danoutils.danocloud.com/apidownload/"
)

// prtHelp prints a short help blurb to standard out
func prtHelp() {
	fmt.Printf("\n----- 'danft' \"Dan File Transfer\" Help -----\n")
	fmt.Printf("01 Use 'danft' to quickly upload or download a file to/from the cloud\n")
	fmt.Printf("02 Upload a file to the cloud: ' danft put <filepath> '\n")
	fmt.Printf("03 Download a file from the cloud: ' danft get <filename> '\n")
	fmt.Printf("04 Download the last file uploaded: ' danft get '\n")
	fmt.Printf("05 NOTE: If your parameters include embedded spaces, remember to enclose those parameters in quotes\n")
	fmt.Printf("----- 'danft' Help -----\n\n")
}

func upload(file string) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := w.CreateFormFile("file", file)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", apiUpEndPt, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("X-Upload-Key", "Jt8iZKQaBphsnpjC")
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}

	return
}

// download downloads a file from the Danocloud
func download(file string) error {
	var (
		err           error
		fname, dFname string
		tURL          string
		req           *http.Request
		res           *http.Response
		dFile         *os.File
		num           int64
	)

	// Missing filename: if yes use "LAST_FILE"
	if len(file) == 0 {
		file = "LAST_FILE"
	}

	// Properly esacape the file value
	fname = url.QueryEscape(file)

	// Construct the request URL
	tURL = apiDnEndPt + fname

	// Now that you have a form, you can submit it to your handler.
	req, err = http.NewRequest("GET", tURL, nil)
	if err != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", err)
		return err
	}

	// Set a header for authentication
	req.Header.Set("X-Upload-Key", "Jt8iZKQaBphsnpjC")

	// Submit the request
	client := &http.Client{}
	res, err = client.Do(req)
	if err != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", err)
		return err
	}

	// Remember to close the response body
	defer res.Body.Close()

	// Get the download file's filename
	if dFname = res.Header.Get("X-Filename"); len(dFname) == 0 {
		// Error occurred - don't have the filename
		log.Printf("ERROR: error occurred fetching the response header. See: %v\n", err)
		return err
	}

	// Create the download file in the local directory
	if dFile, err = os.Create(dFname); err != nil {
		// Error occurred attempting to create a local file (to house the download)
		log.Printf("ERROR: error occurred creating a local file. See: %v\n", err)
		return err
	}

	// Remember to close the file
	defer dFile.Close()

	// Copy the response file data to created file
	if num, err = io.Copy(dFile, res.Body); err != nil {
		// Error occurred attempting to create a local file (to house the download)
		log.Printf("ERROR: error occurred writing the response file data to disk. See: %v\n", err)
		return err
	}

	log.Printf("INFO: wrote %v bytes of file %v to the local disk\n", num, dFname)

	return err
}

func main() {
	// Grab the command line arguments
	args = os.Args

	// Missing command arguments? Print help.
	if len(args) == 1 {
		prtHelp()
		goto WrapUp
	}

	// User explicitly requesting 'help'
	tmp = strings.ToLower(args[1])
	if tmp == "help" || tmp == "h" {
		prtHelp()
		goto WrapUp
	}

	// Validate the non-help operands
	if tmp != "put" && tmp != "get" {
		// Invalid command operand - not "help", "h", "put", or "get"
		fmt.Printf("ERROR: Invalid operand. Use 'help', 'h', 'put', or 'get'\n")
		goto WrapUp
	}

	// Process the command directive
	switch tmp {

	// CASE: download a file
	case "get":
		var gName string
		if len(args) >= 3 {
			// A filename to be "get"ed has been specified
			gName = args[2]
		}

		// Trigger the download
		if err = download(gName); err != nil {
			// Error occurred downloading the file
			fmt.Printf("ERROR: error occurred downloading the file.  See: %v Please try again.\n", err)
			goto WrapUp
		}

	// CASE: upload a file
	case "put":
		// Is there a path or filename operand?
		if len(args) == 2 {
			// Error - don't have a filepath or filename element
			fmt.Printf("ERROR: missing file to be 'put'.  Please try again.\n")
			goto WrapUp
		}

		// Validate that we have a valid path base (e.g. "myfile.txt")
		fpath = args[2]
		fbase = filepath.Base(fpath)

		// Is there a valid base at the end of the path?
		if fbase == "." || fbase == string(os.PathSeparator) {
			// Invalid path base
			fmt.Printf("ERROR: missing or invalid path base.  Please try again.\n")
			goto WrapUp
		}

		// Upload the specified file
		if err = upload(fpath); err != nil {
			fmt.Printf("ERROR: error occurred uploading the specified file.  See: %v\n", err)
			goto WrapUp
		}

		fmt.Printf("INFO: file [%v] has been successfully uploaded.\n", fpath)
	}

WrapUp:
}
