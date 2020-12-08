package main

// This package is an implementation of the command line tool for word searching
// given a bunch of JSON files in a comprssed format(zip). Tool lists out
// matching portions of the text in each file along with file name.
// As of now code is written only to handle JSON format and can be extended.
//
// After reviewing the code, I made the below changes and updated the file.
//
// Review Comments:
// - Deeply nested loops, instead should return early
// - Slicing is prefered than tuple unpacking assignments
// - lac of basic documentation(comments)
// - variable names or too short to understand at first glance
// - lengthy lines (more than 120 chars), wrapping and beatification required
// - excessive use of 'panic()', should handle the errors in higher order functions
// - Modularization and should think of reusability, should write as many functions
//   as possible, and should be open for extention(handling multiple text formats)
// - Idea of this tool has potenital for parallelism, where golang shines
// - command line help should be improved alot and user freidnly, as user has no idea
// - logging should be implemented as user doesn't want to see, dev messages
// - writing basic unittests, ensures the quality
//
// Final thougths:
// I made changes to the main.go file, modularized the code, implemented logging,
// took the advanatge of concurrent design of golang, implemented parallelism, and
// the bench marks are promising. It is nearly 4 times faster than the
// sequential version(depends on core count). We can run parallel version '-j=true' command option.
// Memory mangement, still can be improved. Algorithm can be improved by caching the
// results in a temp file.
//

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// UnStructType interface is to parse unstructured data,
// should go for struct incase of structured data
type UnStructType []map[string]interface{}

// TaskOutput struct is the output sent by paralle tasks
// Info from Processed files is stored in this format
type TaskOutput struct {
	matches []string
	file    string
}

// for Parallel Version
var wg sync.WaitGroup
var mux sync.Mutex

// shared data - atomics
var fileCount uint64
var totalMatches uint64
var errorsWhileProcessing bool // ok with dirty-write

// ParseJSONReader function parses the JSON file and retiurns an interface
// returns nil incase of error
func ParseJSONReader(file *zip.File) UnStructType {

	fileread, err := file.Open()
	if err != nil {
		msg := "Failed to open zip %s for reading: %s"
		log.Fatalln(msg, file.Name, err)
		errorsWhileProcessing = true
		return nil
	}
	defer fileread.Close()

	byteValue, err := ioutil.ReadAll(fileread)
	if err != nil {
		msg := "Failed to read zip %s for reading: %s"
		log.Fatalln(msg, file.Name, err)
		errorsWhileProcessing = true
		return nil
	}

	// UnMarshalling data to JSON
	var all UnStructType
	err = json.Unmarshal(byteValue, &all)

	if err != nil {
		msg := "Failed while unMarshalling: %s"
		errorsWhileProcessing = true
		log.Fatalln(msg, file.Name, err)
		return nil
	}

	return all
}

// FindText function prints partial text containing the 'targetWord' in the given 'text'
// implicitely returns the results
func FindText(text string, targetWord *string, seq *int, taskData *TaskOutput) {
	var textBefore, textAfter string
	words := strings.Split(text, " ")

	for i, w := range words {
		if w == *targetWord {
			*seq++
			start, end := 0, len(words)
			if i-4 > 0 {
				start = i - 4
			}
			textBefore = strings.Join(words[start:i], " ")

			if i+5 <= end {
				end = i + 5
			}
			textAfter = strings.Join(words[i+1:end], " ")
			s := fmt.Sprintf("(%d) %s %s %s\n", *seq, textBefore, *targetWord, textAfter)
			(*taskData).matches = append((*taskData).matches, s)
		}
	}

}

// ParseAndFind (sequential/parallel) functions parses the file to JSON data and performs text searching
// Multiple files can be parsed and searched for the given string
func ParseAndFind(file *zip.File, word *string, parallel *bool) {
	// incase parallel
	if *parallel {
		defer wg.Done()
	}

	jsonData := ParseJSONReader(file)
	taskData := new(TaskOutput)
	if jsonData != nil {
		seq := 0
		for _, _map := range jsonData {
			if text, ok := _map["text"].(string); ok && text != "" {
				FindText(text, word, &seq, taskData)
			}
		}
		if seq > 0 {
			taskData.file = fmt.Sprintf("\t[./%s match(s): %d]\n\n", file.Name, seq)
			PrintOuput(taskData)
		}
	}

}

// PrintOuput function is to print matches and filename as atomic
func PrintOuput(taskData *TaskOutput) {
	for _, str := range (*taskData).matches {
		fmt.Printf(str)
		atomic.AddUint64(&totalMatches, 1)
	}
	fmt.Printf((*taskData).file)
	atomic.AddUint64(&fileCount, 1)
}

// PrintHelp function for user info
func PrintHelp() {
	fmt.Println()
	fmt.Println("|------------------------   Word Searching Program   ---------------------------|")
	fmt.Println("|                                                                               |")
	fmt.Println("|  $ go run main.go -word <word>[mandatory] -zip <file>[optional] -j [optional] |")
	fmt.Println("|  e.g:                                                                         |")
	fmt.Println("|  $ go run main.go -word impressed                                             |")
	fmt.Println("|  $ go run main.go -word impressed -j=true            // to run parallel       |")
	fmt.Println("|  $ go run main.go -word impressed -zip=test.zip       // custom zip           |")
	fmt.Println("|                                                                               |")
	fmt.Println("|-------------------------------------------------------------------------------|")
	fmt.Println()
}

// entry - Let's start here
func main() {
	start := time.Now()
	// command options - word, defaults to null character ("") as targetString
	word := flag.String("word", "", "pass a word to get matching text in all files in a zipped folder")
	// command options - zip, defaults to foc-slack-export.zip
	zipfile := flag.String("zip", "foc-slack-export.zip", "zip file containg JSON files")
	//command options - j, liveraging multiple processors for parallel processing
	parallel := flag.Bool("j", false, "Runs Parallel, Liverages max available cores")

	flag.Parse()

	if *word == "" {
		fmt.Println("Syntx error: <word> required!")
		PrintHelp()
		return
	}

	// setting the basic logger - logs into the file info.log, prevents printing on stdout
	// Can do better logger for production
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer file.Close()

	log.SetOutput(file)

	// creating zip file reader
	rc, err := zip.OpenReader(*zipfile)
	if err != nil {
		log.Fatalln(err)
		return
	}
	defer rc.Close()

	// To livergae max number of processors - for parallel implementation
	runtime.GOMAXPROCS(runtime.NumCPU())

	// walking through files(including sub directories) one at a time (sequentially)
	for _, file := range rc.File {
		if !file.FileInfo().IsDir() && strings.HasSuffix(file.Name, ".json") {
			{
				if *parallel {
					wg.Add(1)
					go ParseAndFind(file, word, parallel)

				} else {
					ParseAndFind(file, word, parallel)
				}
			}
		}
	}

	if *parallel {
		wg.Wait()
	}

	if totalMatches > 0 {
		fmt.Printf("\t%d Matches Found in %d Files\n", totalMatches, fileCount)
	}

	if errorsWhileProcessing {
		fmt.Println("Errors Found While Processing. look at info.log")
	}

	elapsed := time.Since(start)
	fmt.Printf("\tTime Taken %s\n\n", elapsed)
}

// SampleTest Cases

// import "testing"

// func TestFindText(t * testing.T){
// 	// test data
// 	sampleText := "once upona time in India, there was a king called tippu."
// 	targetText = "king"
// 	seq := 0
// 	var taskData TaskOutput

// 	// test scenarios
//  FindText(sampleText, targteWord, &seq, &taskData)
//  if seq != 1 {
//         t.Errorf("FindText() shoudl give 1", seq)
// 	}
//  matchCount := len(taskData.matches)
// 	if matchCount != 1 && taskData.file != nil {
//         t.Errorf("matchCount shoudl be 1", matchCount)
// 	}
// }
