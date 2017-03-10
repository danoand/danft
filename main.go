package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/danoand/utils"
)

var (
	err               error
	tmp, fpath, fbase string
	args              []string

	apiUpEndPt   = "https://danoutils.danocloud.com/uploadafile"
	apiDnEndPt   = "https://danoutils.danocloud.com/apidownload/"
	apiClUpEndPt = "https://danoutils.danocloud.com/pasteaclip"
	apiClDnEndPt = "https://danoutils.danocloud.com/apigetclip"
)

// prtHelp prints a short help blurb to standard out
func prtHelp() {
	fmt.Printf("\n----- 'danft' \"Dan File Transfer\" Help -----\n")
	fmt.Printf("01 Use 'danft' to quickly upload or download a file or 'clip' to/from the cloud\n")
	fmt.Printf("02 Upload a file to the cloud: ' danft put <filepath> '\n")
	fmt.Printf("03 Upload a clip to the cloud: ' danft putclip <string of text in quotes> '\n")
	fmt.Printf("04 Download a file from the cloud: ' danft get <filename> <OPTIONAL new filename>'\n")
	fmt.Printf("05 Download the last file uploaded: ' danft get '\n")
	fmt.Printf("06 Download the last text clip: ' danft getclip '\n")
	fmt.Printf("07 NOTE: If your parameters include embedded spaces, remember to enclose those parameters in quotes\n")
	fmt.Printf("08 NOTE: Use the web application if your clip includes newline characters\n")
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

func uploadclip(clp string) (err error) {
	var fErr error
	var mapClp = map[string]string{"clip": clp}
	var mapStr string

	// Validate the clip data
	if len(clp) == 0 {
		// Error - empty clip
		fmt.Printf("ERROR: your clip has no data.  Try again\n")
		return fmt.Errorf("no clip data")
	}

	// Convert the map to a string value
	if mapStr, _, fErr = utils.ToJSON(mapClp); fErr != nil {
		// Error occurred converting the map object to a string value
		fmt.Printf("ERROR: error converting a map to a string. See: %v\n", fErr)
		return fmt.Errorf("clip data can't be converted to a string")
	}

	// Create a buffer and write the clip data
	var b bytes.Buffer
	b.WriteString(mapStr)

	// Now that you have a form, you can submit it to your handler.
	req, fErr := http.NewRequest("POST", apiClUpEndPt, &b)
	if fErr != nil {
		fmt.Printf("ERROR: error creating an HTTP request. See: %v\n", fErr)
		return fmt.Errorf("error creating an HTTP request")
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("X-Upload-Key", "Jt8iZKQaBphsnpjC")
	req.Header.Set("Content-Type", "application/json")

	// Submit the request
	client := &http.Client{}
	res, fErr := client.Do(req)
	if fErr != nil {
		fmt.Printf("ERROR: error executing the client HTTP request. See: %v\n", fErr)
		return fmt.Errorf("error executing an HTTP request")
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}

	return
}

// download downloads a file from the Danocloud
func download(file, newname string) error {
	var (
		ferr          error
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
	req, ferr = http.NewRequest("GET", tURL, nil)
	if ferr != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", ferr)
		return ferr
	}

	// Set a header for authentication
	req.Header.Set("X-Upload-Key", "Jt8iZKQaBphsnpjC")

	// Submit the request
	client := &http.Client{}
	res, ferr = client.Do(req)
	if ferr != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", ferr)
		return ferr
	}

	// Remember to close the response body
	defer res.Body.Close()

	// Get the download file's filename
	if dFname = res.Header.Get("X-Filename"); len(dFname) == 0 {
		// Error occurred - don't have the filename
		log.Printf("ERROR: error occurred fetching the response header. See: %v\n", ferr)
		return ferr
	}

	// Use a new filename if supplied
	if len(newname) > 1 {
		dFname = newname
	}

	// Create the download file in the local directory
	if dFile, ferr = os.Create(dFname); ferr != nil {
		// Error occurred attempting to create a local file (to house the download)
		log.Printf("ERROR: error occurred creating a local file. See: %v\n", ferr)
		return ferr
	}

	// Remember to close the file
	defer dFile.Close()

	// Copy the response file data to created file
	if num, ferr = io.Copy(dFile, res.Body); ferr != nil {
		// Error occurred attempting to create a local file (to house the download)
		log.Printf("ERROR: error occurred writing the response file data to disk. See: %v\n", ferr)
		return ferr
	}

	log.Printf("INFO: wrote %v bytes of file %v to the local disk\n", num, dFname)

	return ferr
}

// downClip downloads the last clip inserted into the database
func downClip() (string, error) {
	var ferr error
	var tBytes []byte
	var retStr string
	var req *http.Request
	var res *http.Response

	// Create a new HTTP client request
	req, ferr = http.NewRequest("GET", apiClDnEndPt, nil)
	if err != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", ferr)
		return retStr, ferr
	}

	// Set a header for authentication
	req.Header.Set("X-Upload-Key", "Jt8iZKQaBphsnpjC")

	// Submit the request
	client := &http.Client{}
	res, ferr = client.Do(req)
	if ferr != nil {
		log.Printf("ERROR: error occurred creating a new client HTTP request. See: %v\n", ferr)
		return retStr, ferr
	}

	// Remember to close the response body
	defer res.Body.Close()

	if tBytes, ferr = ioutil.ReadAll(res.Body); ferr != nil {
		// Error occurred reading the response contents
		fmt.Printf("ERROR: error occurred readingt the clip response. See: %v\n", ferr)
		return retStr, ferr
	}

	retStr = string(tBytes)

	// Is the response body empty?
	if len(retStr) == 0 {
		return "", nil
	}

	// Have a non-empty response
	return retStr, nil
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
	if tmp != "put" && tmp != "putclip" && tmp != "get" && tmp != "getclip" {
		// Invalid command operand - not "help", "h", "put", or "get"
		fmt.Printf("ERROR: Invalid operand. Use 'help', 'h', 'put', 'putclip' or 'get'\n")
		goto WrapUp
	}

	// Process the command directive
	switch tmp {

	// CASE: download a file
	case "get":
		var gName, diskName string
		if len(args) >= 3 {
			// A filename to be "get"ed has been specified
			gName = args[2]
		}

		if len(args) >= 4 {
			// A new filename has been specified
			diskName = args[3]
		}

		// Trigger the download
		if err = download(gName, diskName); err != nil {
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

	// CASE: upload a clip
	case "putclip":
		// Is there a specified clip?
		if len(args) == 2 {
			// Error - don't have a filepath or filename element
			fmt.Printf("ERROR: missing clip to be uploaded.  Please try again.\n")
			goto WrapUp
		}

		// Upload the specified clip
		if err = uploadclip(args[2]); err != nil {
			fmt.Printf("ERROR: error occurred uploading the specified clip.  See: %v\n", err)
			goto WrapUp
		}

		fmt.Printf("INFO: your clip has been successfully uploaded.\n")

	// CASE: get the last clip
	case "getclip":
		var getClp string

		// Trigger the download
		if getClp, err = downClip(); err != nil {
			// Error occurred downloading a clip
			fmt.Printf("ERROR: Error occurred downloading a clip. See: %v\n", err)
			goto WrapUp
		}

		if len(getClp) == 0 {
			// Empty clip
			fmt.Printf("INFO: Your clip is empty or blank.\n")
			goto WrapUp
		}

		// Print out the clip
		fmt.Printf("------------\n")
		fmt.Printf("%v", getClp)
		fmt.Printf("------------\n")
	}

WrapUp:
}
