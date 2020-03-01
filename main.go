package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const ConfName = "conf.json"

var (
	h    bool
	c    string
	conf Conf
)

func main() {
	flag.Parse()

	if h {
		flag.Usage()
		return
	}

	parseConf()
	http.HandleFunc(conf.Url, proxyHandler)
	http.ListenAndServe(":"+conf.Port, nil)
}

func init() {
	currDir, _ := os.Getwd()
	flag.BoolVar(&h, "h", false, "help")
	flag.StringVar(&c, "c", currDir+string(os.PathSeparator)+ConfName, "Set Configuration file")

	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stdout, `picBed proxy for MWeb usage github 
Usage: githubPicBedProxy [-c filePath]

Options:
`)
	flag.PrintDefaults()
}

func parseConf() {
	filePtr, err := os.Open(c)
	if err != nil {
		fmt.Printf("Open file failed [Err:%s]\n", err.Error())
		panic(err)
	}
	defer filePtr.Close()

	err = json.NewDecoder(filePtr).Decode(&conf)
	if err != nil {
		fmt.Printf("Decoder failed [Err:%s]\n", err.Error())
		panic(err)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	file,_,_ := r.FormFile(conf.ParamName)
	bytes,err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("ParseForm failed [Err:%s]\n", err.Error())
		response(w,false,"ParseForm error","")
		return
	}

	putToGithub(w,bytes)
}

func putToGithub(w http.ResponseWriter,bs []byte) {
	timestamp := time.Now().Format("20060102150405")
	path := timestamp + ".jpg"
	githubUrl := "https://api.github.com/repos/" + conf.Repo + "/contents/" + path
	encodingContent := base64.StdEncoding.EncodeToString(bs)
	uploadReq := GitHubUploadReq{
		Branch:conf.Branch,
		Message:"upload file " + timestamp,
		Content:encodingContent,
		Path:path,
	}

	putBytes,err := json.Marshal(uploadReq)
	if err != nil {
		fmt.Printf("putToGithub failed [Err:%s]\n", err.Error())
		response(w,false,"putToGithub error","")
		return
	}


	req,_ :=http.NewRequest(http.MethodPut,githubUrl,bytes.NewReader(putBytes))
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Authorization","token "+conf.Token)

	res,err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("putToGithub failed [Err:%s]\n", err.Error())
		response(w,false,"putToGithub error","")
		return
	}
	defer res.Body.Close()
	respBody,_ := ioutil.ReadAll(res.Body)

	var respInfo map[string]map[string]interface{}
	json.NewDecoder(bytes.NewBuffer(respBody)).Decode(&respInfo)
	downloadUrl := respInfo["content"]["download_url"]
	fmt.Println("downloadUrl:",downloadUrl)
	response(w,true,"success",downloadUrl.(string))
}

func response(w http.ResponseWriter,succ bool,info string,url string) {
	resp := MWebResp{
		DownloadUrl:url,
		Info:info,
	}

	if succ {
		resp.Status = "success"
	} else {
		resp.Status = "false"
	}

	b,_ := json.Marshal(&resp)
	w.Header().Add("Content-Type","text/json;charset=utf-8")
	w.Write(b)

}

type Conf struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
	Token  string `json:"token"`

	Port string `json:"port"`
	Url  string `json:"url"`
	ParamName string `json:"paramName"`
}

type GitHubUploadReq struct {
	Branch string `json:"branch"`
	Message string `json:"message"`
	Content string `json:"content"`
	Path string `json:"path"`
}

type MWebResp struct {
	Status      string `json:status`
	DownloadUrl string `json:downloadUrl`
	Info string `json:"info"`
}
