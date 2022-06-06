package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	inputFilePath  string
	outputFilePath string
)

func parseArgs() {
	// TODO: Optimize code
	flag.StringVar(&inputFilePath, "i", "", "input file")
	flag.StringVar(&outputFilePath, "o", "", "output file")
	flag.Parse()

	if inputFilePath == "" {
		if len(flag.Args()) != 0 {
			inputFilePath = flag.Args()[0]
		} else {
			log.Fatal("no input file given or not a html file")
		}
	}
	if outputFilePath == "" {
		if len(flag.Args()) > 2 {
			outputFilePath = flag.Args()[1]
		} else {
			outputFilePath = strings.Replace(inputFilePath, ".html", ".compat.html", 1)
		}
	}
}

func handleReg(result []string) (string, error) {
	_, capturedStr := result[0], result[1]

	if strings.HasPrefix(capturedStr, "http") {
		resp, err := http.Get(capturedStr)
		if err != nil {
			log.Println("can not download img file: ", err, capturedStr)
			return "", err
		}
		defer resp.Body.Close()
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("can not read img file: ", err, capturedStr)
			return "", err
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType == "" {
			log.Println("Failed to get Content-Type: ", capturedStr)
			return "", err
		}

		contentBase64 := base64.StdEncoding.EncodeToString(content)
		encodeResult := fmt.Sprintf("<img src=\"data:%s;base64,%s\"", contentType, contentBase64)

		return encodeResult, nil

	} else if !strings.HasPrefix(capturedStr, "data:image/") {
		var imgFilePath string
		if filepath.IsAbs(capturedStr) {
			imgFilePath = capturedStr
		} else {
			dir, _ := filepath.Split(inputFilePath)
			imgFilePath = dir + capturedStr
		}

		res := strings.Split(capturedStr, ".")
		if len(res) == 0 {
			log.Println("can not find img ----")
		}

		contentType := res[len(res)-1]

		imgFile, err := os.OpenFile(imgFilePath, os.O_RDONLY, 0644)
		if err != nil {
			log.Println("can not open img file: ", err, capturedStr)
			return "", err
		}
		defer imgFile.Close()

		content, err := ioutil.ReadAll(imgFile)
		if err != nil {
			log.Println("can not read img file: ", err, imgFilePath)
			return "", err
		}

		contentBase64 := base64.StdEncoding.EncodeToString(content)
		encodeResult := fmt.Sprintf("<img src=\"data:%s;base64,%s\"", contentType, contentBase64)

		return encodeResult, nil
	}
	return "", errors.New("unkonwn img")
}

func handleFile() {
	var newContent string

	oldFile, err := os.OpenFile(inputFilePath, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal("can not open input html file: ", err, inputFilePath)
	}
	defer oldFile.Close()

	oldContentBytes, err := ioutil.ReadAll(oldFile)
	if err != nil {
		log.Fatal("can not read input html file: ", err, inputFilePath)

	}

	oldContent := string(oldContentBytes)
	newContent = oldContent

	reg := regexp.MustCompile(`<img src="(.*?)"`)
	if reg == nil {
		log.Fatal("正则解析失败")
	}

	results := reg.FindAllStringSubmatch(oldContent, -1)
	var mux sync.Mutex
	for _, result := range results {
		go func(result []string) {
			res, err := handleReg(result)
			if err != nil {
				return
			}
			mux.Lock()
			defer mux.Unlock()
			newContent = strings.Replace(newContent, result[0], res, 1)
		}(result)
	}

	outputFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("can not open output html file: ", err, outputFilePath)
	}
	defer outputFile.Close()
	_, err = outputFile.Write([]byte(newContent))
	if err != nil {
		log.Fatal("can not write to output html file: ", err, outputFilePath)

	}

}

func main() {

	parseArgs()
	handleFile()

}
