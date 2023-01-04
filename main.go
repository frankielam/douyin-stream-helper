package main

import (
    "fmt"
    "log"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "sync"
    "os/exec"
    "time"
    "strconv"
    "crypto/sha256"
    "strings"
)

var VIDEO_TIME_LENGTH int = 300 // 600 => 60*10 second
// var FILE_PATH = "/Volumes/Usb/ffmpeg_container/"
var FILE_PATH = "/app/out/"

type Data struct {
    Title string
    Url   string
    Key string
}

type FfmpegServer struct {
    Message chan Data
    mapLock sync.RWMutex
    Count int64
    ProcessingMap map[string]int64
}

func (this *FfmpegServer) enqueue(msg *Data) {
    // fmt.Println(msg.Title)
    // fmt.Println(msg.Url)
    this.Count++
    this.Message <- *msg
}

func (this *FfmpegServer) Handle(w http.ResponseWriter, r *http.Request) {
    // r.ParseForm()
    log.Println("path", r.URL.Path)

    timeNow := time.Now().Unix()

    var payload Data
    body, _ := ioutil.ReadAll(r.Body)
    err := json.Unmarshal(body, &payload)
    if err != nil {
        log.Fatal("Error during Unmarshal(): ", err)
    }

    s := r.URL.Query().Get("s")
    t := r.URL.Query().Get("t")

    time, _ :=  strconv.ParseInt(t, 10, 64)
    // fmt.Println("time diff", timeNow - time)
    hash := sha256.New()
    str := payload.Url + t
    hash.Write([]byte(str))
    sign := fmt.Sprintf("%x", hash.Sum(nil))


    if sign != s || time > timeNow || timeNow - time > 5 {
        if sign != s {
            log.Println("ERROR: signature invalid") 
        }
        if time > timeNow {
            log.Println("ERROR: time > timeNow")    
        }
        if timeNow - time > 5 {
            log.Println("ERROR: timeNow - time is  ", timeNow - time)   
        }
        
        fmt.Fprintf(w, "error")
        return 
    }


    
    go this.enqueue(&payload)
    fmt.Fprintf(w, "ok")
}

func (this *FfmpegServer) Listener() {
    log.Println("Listener is starting")
    for {
        msg := <- this.Message
        fmt.Println(this.Count)
        fmt.Println(msg)
        
        hash := sha256.New()
        hash.Write([]byte(msg.Url))
        key := fmt.Sprintf("%x", hash.Sum(nil))

        val, exists := this.ProcessingMap[key]; 
        if exists {
            fmt.Printf("WARNING\tURL[%s] is handling (%d)\n", msg.Url, val)
            continue
        }
        this.mapLock.Lock()
        this.ProcessingMap[key] = time.Now().Unix()
        this.mapLock.Unlock()
        var counter int = 1
        go this.CallSysCmd(key, msg, counter)
    }
}

// func (this *FfmpegServer) SaveStream(key string, obj Data) {
//     var counter int = 1
//     this.CallSysCmd(key, obj, counter)
// }

func (this *FfmpegServer) CallSysCmd(key string, obj Data, counter int) {
    if counter >= 30 {
        return 
    }
    command, errs := exec.LookPath("ffmpeg")
    fmt.Println(command)
    if errs != nil {
        // panic(err)
        fmt.Println("$$$$$$$$$$$$$")
    }

    currentTime := time.Now()  

    timestamp := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
    currentTime.Year(), currentTime.Month(), currentTime.Day(),
    currentTime.Hour(), currentTime.Minute(), currentTime.Second())
  
    filename := obj.Title + "-" + timestamp + ".mp4"
    args := []string{"-i", obj.Url}
    // args = append(args, "-s", "720*1280")
    // args = append(args, "-s", "480*854")
    args = append(args, "-s", "360*640")
    // args = append(args, "-c", "copy")
    args = append(args, "-t", strconv.Itoa(VIDEO_TIME_LENGTH))
    // args = append(args, "-rw_timeout", "3000000")
    args = append(args, "-ac", "1")
    args = append(args, "-ar", "8000")
    millstime := fmt.Sprintf("%d000000", currentTime.Unix())
    args = append(args, "-filter_complex", "drawtext=expansion=strftime:basetime="+ millstime +" :text='%Y-%m-%d %H\\:%M\\:%S' : fontsize=80 : fontcolor=red : fontfile=/app/Arial.ttf")

    args = append(args, FILE_PATH + filename)

    fmt.Println(args)

    cmd := exec.Command("ffmpeg", args...)

    // stdout, err := cmd.StdoutPipe()
    stdout, err := cmd.StderrPipe()

    if err != nil {
        // log.Fatal(err)
        log.Println(err)
    }
    
    if err := cmd.Start(); err != nil {
        log.Println(err)
    }
    log.Println("SUB PROCESS #", cmd.Process.Pid, "\t", filename)
    data, err := ioutil.ReadAll(stdout)

    if err != nil {
        // log.Fatal(err)
        log.Println(cmd.Process.Pid, err)
    }

    if err := cmd.Wait(); err != nil {
        log.Println("$$$$$$$$$$$$$$$$$$$$", cmd.Process.Pid)
        // log.Fatal(err)
        log.Println(err)
        log.Println("|", err.Error(), "|")
        if strings.Contains(err.Error(), "exit status 1") {
            log.Printf("%d DONE!",  cmd.Process.Pid)
            this.mapLock.Lock()
            delete(this.ProcessingMap, key)
            this.mapLock.Unlock()
            return 
        }
    }

    str := string(data)
    
    log.Printf("%s\n", len(str))

    // var out, outErr bytes.Buffer 
    // cmd.Stdout = &out
    // cmd.Stderr = &outErr
    // err = cmd.Run()
    
    // if err != nil {
    //     fmt.Println("$$$$$$$$$$$$$$$$$$$$")
    //     log.Fatal("ERR:", err)
    // }


    // fmt.Printf("command output: %s\n", out.String())
    // fmt.Printf("command err output: %s\n", outErr.String())
    log.Println("############# Next time ##############\t", counter + 1 )
    go this.CallSysCmd(key, obj, counter + 1)
}

func (this *FfmpegServer) Start() {
    port := "9999"
    go this.Listener()
    http.HandleFunc("/22a4c19296767a3dcb03a6c516ffbeb82f35c3a6848556c598e8d6fb7c5c3415", this.Handle) 
    log.Println("http://127.0.0.1:" + port)
    err := http.ListenAndServe(":" + port, nil)
    if err != nil {
        log.Fatal("Listen And Serve:", err)
    }
}

func main() {
    service := &FfmpegServer {
        Message: make(chan Data),
        Count: 0,
        ProcessingMap: make(map[string]int64),
    }
    service.Start()
}
