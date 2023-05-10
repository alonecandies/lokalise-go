package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v52/github"
	// "github.com/lokalise/go-lokalise-api/v3"
	"golang.org/x/oauth2"
)

const (
	owner    = "alonecandies"
	repo     = "lokalise-go"
	basePath = "locale"
)

var client *github.Client

func calculateGitSHA1(contents []byte) []byte {
	contentLen := len(contents)
	blobSlice := []byte("blob " + strconv.Itoa(contentLen))
	blobSlice = append(blobSlice, '\x00')
	blobSlice = append(blobSlice, contents...)
	h := sha1.New()
	h.Write(blobSlice)
	bs := h.Sum(nil)
	return bs
}

func getContents(ctx context.Context, path string) {
	fileContent, directoryContent, resp, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%#v\n", fileContent)
	fmt.Printf("%#v\n", directoryContent)
	fmt.Printf("%#v\n", resp)

	for _, c := range directoryContent {
		fmt.Println(*c.Type, *c.Path, *c.Size, *c.SHA)

		local := filepath.Join(basePath, *c.Path)
		fmt.Println("local:", local)

		if *c.Name == "locale" {
			switch *c.Type {
			case "file":
				_, err := os.Stat(local)
				if err == nil {
					b, err1 := ioutil.ReadFile(local)
					if err1 == nil {
						sha := calculateGitSHA1(b)
						if *c.SHA == hex.EncodeToString(sha) {
							fmt.Println("no need to update this file, the SHA is the same")
							continue
						}
					}
				}
				downloadContents(ctx, c, local)
			case "dir":
				getContents(ctx, filepath.Join(path, *c.Path))
			}
		}

	}
}

func downloadContents(ctx context.Context, content *github.RepositoryContent, localPath string) {
	if content.Content != nil {
		fmt.Println("content:", *content.Content)
	}

	rc, _, err := client.Repositories.DownloadContents(ctx, owner, repo, *content.Path, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.MkdirAll(filepath.Dir(localPath), 0666)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Writing the file:", localPath)
	f, err := os.Create(localPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	n, err := f.Write(b)
	if err != nil {
		fmt.Println(err)
	}
	if n != *content.Size {
		fmt.Printf("number of bytes differ, %d vs %d\n", n, *content.Size)
	}
}

func main() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: os.Getenv("GITHUB_TOKEN"),
		},
	)
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)
	getContents(ctx, "")
}
