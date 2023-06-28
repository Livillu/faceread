package main

import (
  "bytes"
  "crypto/aes"
  "crypto/cipher"
  "database/sql"
  "encoding/base64"
  "encoding/json"
  "faceread/libdata"
  "flag"
  "fmt"
  _ "github.com/denisenkom/go-mssqldb"
  "github.com/gin-gonic/gin"
  "github.com/gorilla/websocket"
  "github.com/kardianos/service"
  "net/http"
  //"golang.org/x/sys/windows"
  "golang.org/x/sys/windows/registry"
  "html/template"
  "io/ioutil"
  "log"
  "math/rand"
  "os"
  "strconv"
  "strings"
  "sync"
  "time"
)

type TimesGo struct {
  Days    int
  Hours   int
  Minutes int
}

var TextMessage = 1
var BinaryMessage = 2
var CloseMessage = 8
var PingMessage = 9
var PongMessage = 10

var mutex sync.Mutex

var MssqlConnection18 = "server=192.168.8.18;user id=sa;password=sa!QA@WS;database=SmartCard"

//var MssqlConnection18 = "" //server=192.168.8.18;user id=sa;password=sa!QA@WS;database=SmartCard"
var ZoneTime = "Myanmar,6:30" //仰光
//var ZoneTime = "Cambodia,7:0"    //金邊
//var ZoneTime = "UTC+8" //台灣

var TimesGos []TimesGo //定時抓檔時間
var OnceHours []string //定時抓檔開始往前多少小時

var eveTimes = 0           //週期抓檔多少分鐘抓一次
var BefoMinutes = "0"      //週期抓檔開始往前多少分鐘
var ThisTime *time.Time    //現在時間
var FaceCon *libdata.Sinfs //Client連線通道
var Restart = ":"

//啟動
var upgrader = websocket.Upgrader{}

//編碼---------------------------------------------------
func Ecodecs(text string) string {
  result := base64.StdEncoding.EncodeToString([]byte(text))
  return string(result)
}

func Dcodecs(text string) string {
  result, _ := base64.StdEncoding.DecodeString(text)
  return string(result)
}

func winregisty() string {
  k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.SET_VALUE)
  if err != nil {
    log.Fatal(err)
  }
  defer k.Close()

  s, _, err := k.GetStringValue("facedb")
  if err != nil {
    log.Fatal(err)
  }
  if s == "" {
    k.SetStringValue("facedb", "server=192.168.8.18;user id=sa;password=sa!QA@WS;database=SmartCard")
  }
  k.Close()
  fmt.Printf("Windows system root is %q\n", s)
  return s
}

func winsetregisty(si string) {
  k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE|registry.SET_VALUE)
  if err != nil {
    log.Fatal(err)
  }
  defer k.Close()

  si1, _, err := k.GetStringValue("facedb")
  if err != nil {
    log.Fatal(err)
  }
  if si1 != si {
    k.SetStringValue("facedb", si)
  }
  k.Close()
  fmt.Printf("Windows system facedb is %q\n", si)

}
func AesEncrypt(orig string) string {
  // 轉成字節數組
  origData := []byte(orig)
  k := []byte("0123456789012345")
  // 分組祕鑰
  // NewCipher該函數限制了輸入k的長度必須爲16, 24或者32
  block, _ := aes.NewCipher(k)
  // 獲取祕鑰塊的長度
  blockSize := block.BlockSize()
  // 補全碼
  origData = PKCS7Padding(origData, blockSize)
  // 加密模式
  blockMode := cipher.NewCBCEncrypter(block, k[:blockSize])
  // 創建數組
  cryted := make([]byte, len(origData))
  // 加密
  blockMode.CryptBlocks(cryted, origData)
  return base64.StdEncoding.EncodeToString(cryted)
}

func AesDecrypt(cryted string) string {
  // 轉成字節數組
  crytedByte, _ := base64.StdEncoding.DecodeString(cryted)
  k := []byte("0123456789012345")
  // 分組祕鑰
  block, _ := aes.NewCipher(k)
  // 獲取祕鑰塊的長度
  blockSize := block.BlockSize()
  // 加密模式
  blockMode := cipher.NewCBCDecrypter(block, k[:blockSize])
  // 創建數組
  orig := make([]byte, len(crytedByte))
  // 解密
  blockMode.CryptBlocks(orig, crytedByte)
  // 去補全碼
  orig = PKCS7UnPadding(orig)
  return string(orig)
}

//補碼
//AES加密數據塊分組長度必須爲128bit(byte[16])，密鑰長度可以是128bit(byte[16])、192bit(byte[24])、256bit(byte[32])中的任意一個。
func PKCS7Padding(ciphertext []byte, blocksize int) []byte {
  padding := blocksize - len(ciphertext)%blocksize
  padtext := bytes.Repeat([]byte{byte(padding)}, padding)
  return append(ciphertext, padtext...)
}

//去碼
func PKCS7UnPadding(origData []byte) []byte {
  length := len(origData)
  unpadding := int(origData[length-1])
  return origData[:(length - unpadding)]
}

//設定檔解碼
func resetconfig(encodestring string) {
  isok := false
  bytes, err := ioutil.ReadFile("webconfig.ini")
  check(err)
  w := strings.Split(string(bytes), "\r\n") //所有行
  for index, row := range w {               //每一行
    if len(row) > 6 {
      s := strings.SplitN(row, "#", 2)
      if len(s) == 2 && s[0] == "Connection1" {
        w[index] = "Connection1#" + encodestring
        isok = true
      }
    }
  }
  if isok {
    contex := strings.Join(w, "\r\n")
    buf := []byte(contex)
    fo, err := os.OpenFile("webconfig.ini", os.O_WRONLY, 0660)
    if err != nil {
      panic(err)
    }
    defer func() {
      if err := fo.Close(); err != nil {
        panic(err)
      }
    }()
    if _, err := fo.Write(buf[:]); err != nil {
      panic(err)
    }
    fmt.Println(contex)
  }
}

//----------------------------------------

//異常處理
func check(e error) bool {
  if e != nil {
    fmt.Print(e)
    return false
  } else {
    return true
  }
}

//寫入日誌檔
func WriteLog(msg string) {
  t := time.Now()
  T := t.Format("20060102") + ".log"
  f, err := os.OpenFile(T, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    log.Fatal(err)
  }
  if _, err := f.Write([]byte(msg + "\r\n")); err != nil {
    log.Fatal(err)
  }
  if err := f.Close(); err != nil {
    log.Fatal(err)
  }
}

//自動轉檔設定檔初始化
func loadconfig() {
  bytes, err := ioutil.ReadFile("webconfig.ini")
  check(err)
  w := strings.Split(string(bytes), "\r\n")
  for _, row := range w {
    if len(row) > 6 {
      s := strings.SplitN(row, "#", 2)
      if len(s) == 2 {
        s[1] = strings.TrimSpace(s[1])
        switch s[0] {
        case "Connection1": //連線
          MssqlConnection18 = s[1]
        case "ZoneTime": //時區
          ZoneTime = s[1]
          fmt.Println("ZoneTime:", s[1])
        case "EveTimes": //連續抓資料
          eveTimes, err = strconv.Atoi(s[1])
          if err != nil {
            fmt.Println("EveTimes Err:", err)
          } else {
            fmt.Println("EveTimes:", eveTimes)
          }
        case "BefoMinutes": //連續抓資料往前抓
          BefoMinutes = s[1]
          if err != nil {
            fmt.Println("BefoMinutes Err:", err)
          } else {
            fmt.Println("BefoMinutes:", BefoMinutes)
          }
        case "Restart": //連續抓資料往前抓
          Restart = s[1]
          if err != nil {
            fmt.Println("Restart Err:", err)
          } else {
            fmt.Println("Restart:", Restart)
          }
        case "OnceTime": //定時抓資料
          fmt.Println("OnceTime:", s[1])
          s0 := strings.Split(s[1], ",")
          for _, s01 := range s0 {
            if len(s01) > 4 {
              s1 := strings.Split(s01, "-")
              var TG TimesGo
              TG.Days, err = strconv.Atoi(s1[0])
              TG.Hours, err = strconv.Atoi(s1[1])
              TG.Minutes, err = strconv.Atoi(s1[2])
              TimesGos = append(TimesGos, TG)
            }
          }
        case "OnceHours": //定時抓資料往前抓
          fmt.Println("OnceHours:", s[1])
          s1 := strings.Split(s[1], ",")
          for _, s11 := range s1 {
            if len(s11) > 0 {
              OnceHours = append(OnceHours, "-"+s11+"h")
            }
          }
        }
      }
    }
  }
}

//string Time轉Unix time
func sTot(stime string) int64 {
  LsdateTime := strings.Split(stime, " ")
  yyyy, _ := strconv.Atoi(LsdateTime[0][:4])
  MM, _ := strconv.Atoi(LsdateTime[0][5:7])
  dd, _ := strconv.Atoi(LsdateTime[0][8:10])
  Lstime := strings.Split(LsdateTime[1], ":")
  hh, _ := strconv.Atoi(Lstime[0])
  mm, _ := strconv.Atoi(Lstime[1])

  arear := strings.Split(ZoneTime, ",")
  ttime := strings.Split(arear[1], ":")
  h1, _ := strconv.Atoi(ttime[0])
  h2, _ := strconv.Atoi(ttime[1])
  utc := time.FixedZone(arear[1], (h1*60*60)+(h2*60))

  month := time.Month(MM)
  t := time.Date(yyyy, month, dd, hh, mm, 0, 0, utc)
  t1 := t.Unix()
  return t1
}

//刷臉機連線時註冊PORT
func InsertSinf(ip string, c *websocket.Conn, FaceCon *libdata.Sinfs) libdata.Sinf {
  LSinf := libdata.Sinf{}
  Tc := *FaceCon
  for index, row := range Tc.FaceCons {
    if ip == row.Ip {
      var time1 = time.Now()
      tsecomd := time1.Unix()
      Recv := make(chan []byte)
      Tc.FaceCons[index].FaceCon = c
      Tc.FaceCons[index].Recv = Recv
      Tc.FaceCons[index].Tsecomd = tsecomd
      msg := "Add Con:" + Tc.FaceCons[index].Id + "  IP:" + Tc.FaceCons[index].Ip + "  Name:" + Tc.FaceCons[index].Name + "  Time:" + time1.Format("2006.01.02 15:04:05")
      fmt.Println(msg)
      WriteLog(msg)
      LSinf = Tc.FaceCons[index]
      break
    }
  }
  *FaceCon = Tc
  return LSinf
}

//個別更新資料
//使用者ID，廠別，JPG[]
//
func restpic(cardid, factory string, jpgList []string) {
  db, err := sql.Open("mssql", MssqlConnection18)
  if err != nil {
    fmt.Println("err:", err)
  }
  tfaceCon := *FaceCon
  c1, msg := tfaceCon.SearchSinf(cardid)
  if msg == "" {
    for _, jpgList1 := range jpgList {
      jpg := strings.Split(jpgList1, ".")
      defer func() {
        db.Close()
        recover()
      }()
      var fp_id, emp_fname, sex, brdate, photo string
      sqlstring := "SELECT [MemberNo],[MemberName] ,case [Sex] when '1' then 'female' else 'male' end  [Sex]"
      sqlstring += ",[Birthday],[MemberPic] FROM [SmartCard].[dbo].[Member_Info] where [MemberNo]=$1"
      db.QueryRow(sqlstring, jpg[0]).Scan(&fp_id, &emp_fname, &sex, &brdate, &photo)
      brdate = brdate[:10]
      photo = "http://192.168.20.254:8083/res/photo2/" + factory + "/" + jpgList1
      fmt.Println("photo:", photo)
      //--------------------------------
      var personrow libdata.AddPerson
      personrow.Id = rand.Intn(100)
      personrow.Method = "personnelData.savePersons"
      var persons libdata.Persons
      persons.Person.Type = 1                                //人员类型,1-内部员工,2-访客, 3-黑名单；必填
      persons.Person.Code = fp_id                            //人员编号；必填
      persons.Person.Name = emp_fname                        //姓名；必填
      persons.Person.Sex = sex                               //性别；male/female；必填
      persons.Person.Birthday = brdate                       //生日；必填
      persons.Person.URL = append(persons.Person.URL, photo) //URL图片导入
      personrow.Params = append(personrow.Params, persons)
      res2B, _ := json.Marshal(personrow)
      fmt.Println(string(res2B))
      err := c1.FaceCon.WriteJSON(personrow)
      if err != nil {
        log.Println("write SendPersons:", err)
      } else {
        data1 := <-c1.Recv
        fmt.Println(string(data1))
      }
    }
  }
}

//取得資料庫內機台，部門 LIST
func Getselects(c *gin.Context) {
  cardId := c.PostForm("cardId")
  db, err := sql.Open("mssql", MssqlConnection18)
  if err != nil {
    fmt.Println("err:", err)
  }
  defer func() {
    db.Close()
    recover()
  }()
  var Ctmps []libdata.Clock
  var Dtmps []libdata.Clock
  sqlwhere := ""
  if cardId != "" {
    sqlwhere = " where AccessControlNo='" + cardId + "'"
  }
  rows, err := db.Query("SELECT distinct [AccessControlNo],[AccessControlName] FROM [SmartCard].[dbo].[AccessControl_Info]" + sqlwhere)
  if err != nil {
    fmt.Println(err.Error())
  } else {
    var clock_id, clock_name string
    for rows.Next() {
      rows.Scan(&clock_id, &clock_name)
      Ctmps = append(Ctmps, libdata.Clock{clock_id, clock_name, ""})
    }
  }
  rows.Close()
  sqlstring := "SELECT [ID],[OrgName] FROM [SmartCard].[dbo].[Org_Info] where [ParentNode]='1' and [IsEnabled]='1'"
  rows, err = db.Query(sqlstring)
  if err != nil {
    fmt.Println(err.Error())
  } else {
    var clock_id, clock_name string
    for rows.Next() {
      rows.Scan(&clock_id, &clock_name)
      Dtmps = append(Dtmps, libdata.Clock{clock_id, clock_name, ""})
    }
  }
  c.JSON(200, gin.H{"Clocks": Ctmps, "Depart": Dtmps})
}

//刷臉機刷卡資料匯入DATABASE
func TimesDatabase(records libdata.GetData, clock_id string) {
  db, err := sql.Open("mssql", MssqlConnection18)
  if err != nil {
    fmt.Println("err:", err)
  }
  defer func() {
    db.Close()
    recover()
  }()
  sqlstring := "if (select count(*) from [SmartCard].[dbo].[FaceRecord_Info] where [AccessControlNo]=$1 and [PassTime]=$2 and [MemberID]=$3 and Pass=1)=0 "
  sqlstring += "INSERT INTO [SmartCard].[dbo].[FaceRecord_Info] ([AccessType],[CardType],[Detail],[RecordID],[Pass],[IsEnabled],[CreateDate],[Remarks],"
  sqlstring += "[MemberID],[AccessControlNo],[PersonCode],[PassTime],[PersonName]) "
  sqlstring += "VALUES('Face',1,'Successful passage',1,1,1,getdate(),'OK',$4,$5,$6,$7,(SELECT TOP (1) [MemberName] FROM [SmartCard].[dbo].[Member_Info] where [MemberNo]=$8))"
  for _, row := range records.Params.Records {
    Ttime := (time.Unix(row.Time, 0)).Format("2006-01-02 15:04:05")
    row1, err := db.Exec(sqlstring, clock_id, Ttime, row.PersonCode, row.PersonCode, clock_id, row.PersonCode, Ttime, row.PersonCode)
    if err != nil {
      fmt.Println(sqlstring, clock_id, Ttime, row.PersonCode, row.PersonCode, clock_id, row.PersonCode, Ttime, row.PersonCode)
      log.Fatal(err)
    } else {
      val, err := row1.RowsAffected()
      if err != nil {
        fmt.Println("Err msg:", err)
      } else {
        if val > 0 {
          fmt.Println(clock_id, Ttime, row.PersonCode)
        }
      }
    }
  }
}

//伺服器與刷臉機連線通道
func ReadRcv(c1 libdata.Sinf) {
  for {
    conn := c1.FaceCon
    messageType, p, err := conn.ReadMessage()
    if err != nil {
      WriteLog("remove>> Ip:" + c1.Ip + time.Now().Format("2006.01.02 15:04:05"))
      log.Println(c1.Ip, "Off Line:", err)
      faceCon := *FaceCon
      *FaceCon = faceCon.RemoveSinf(c1.Ip)
      return
    }
    if messageType == TextMessage && string(p) == "ping" {
      err = conn.WriteMessage(TextMessage, []byte("pong"))
      if err != nil {
        log.Println("write:", err)
      }
    } else if messageType == TextMessage && string(p) != "ping" {
      c1.Recv <- p
    }
  }
}

//取得所有刷臉機資訊
func getAccessControl() *[]libdata.AccessControl {
  var tmps []libdata.AccessControl
  var IsNotCon = true
  if IsNotCon {
    sqlstring := "SELECT [AccessControlNo],[AccessControlName],[IP] FROM [AccessControl_Info] order by IP"
    db, err := sql.Open("mssql", MssqlConnection18)
    defer func() {
      db.Close()
      recover()
      time.Sleep(10 * time.Second)
    }()
    if err == nil {
      rows, err1 := db.Query(sqlstring)
      if err1 != nil {
        fmt.Println("err:MssqlConnection err")
      } else {
        var AccessControlNo, AccessControlName, IP sql.NullString
        for rows.Next() {
          rows.Scan(&AccessControlNo, &AccessControlName, &IP)
          tmps = append(tmps, libdata.AccessControl{AccessControlNo.String, IP.String, AccessControlName.String, "Off"})
        }
        IsNotCon = false
      }
    }
  }
  return &tmps
}

//
//單機刷臉機資料上傳

func machOneUpdate1A(c1 libdata.Sinf, st, ed int64, sno int) {
  //stting Time轉Unix time
  counts := 1000
  i := 0
  var record = libdata.GetData{}
  currentTime := time.Now()
  msg := "ID:" + c1.Id + "  IP:" + c1.Ip + "  Name:" + c1.Name + "  Time:" + currentTime.Format("2006.01.02 15:04:05") + ": Update is start.....\r\n"
  fmt.Println(msg)
  for counts == 1000 {
    mdata := libdata.SelectData{}
    mdata.Id = rand.Intn(1000)
    mdata.Method = "accessRecord.find"
    //mdata.Params.Condition.PersonCode = ""
    mdata.Params.Condition.StartTime = st
    mdata.Params.Condition.EndTime = ed
    mdata.Params.Condition.AccessType = "Face"
    mdata.Params.Condition.Offset = i * 1000
    mdata.Params.Condition.Limit = 1000
    //4.SEARCH人員刷卡紀錄
    //c1.Recv = make(chan []byte)
    err := c1.FaceCon.WriteJSON(mdata)
    if err == nil {
      var record1 = libdata.GetData{}
      data1 := <-c1.Recv
      err = json.Unmarshal(data1, &record1)
      if err == nil {
        counts = len(record1.Params.Records)
        if counts > 0 {
          if len(record.Params.Records) == 0 {
            record.Id = record1.Id
            record.Method = record1.Method
          }
          for _, row := range record1.Params.Records {
            if row.Pass {
              record.Params.Records = append(record.Params.Records, row)
            }
          }
        }
        fmt.Println(c1.Ip, ":Update is ok ...")
        break
      }
    } else {
      fmt.Println(c1.Ip, ":Update failed !!!!")
      break
    }
  }
  fmt.Println("Records:", len(record.Params.Records))
  msg += "Records:" + strconv.Itoa(len(record.Params.Records)) + "\r\n"
  if len(record.Params.Records) > 0 {
    TimesDatabase(record, c1.Id)
  }
  currentTime = time.Now()
  fmt.Println(c1.Id, c1.Name, currentTime.Format("2006.01.02 15:04:05"), ": Update is end.....")
  msg += c1.Id + "  " + c1.Name + "  " + currentTime.Format("2006.01.02 15:04:05") + ": Update is end.....\r\n"
  WriteLog(msg)
}

//WEB服務入口程式
func web(c *gin.Context) {
  t, _ := template.ParseFiles("websocket.html")
  t.Execute(c.Writer, nil)
}

//取得部門員list
func Getempoeey(c *gin.Context) {
  var Etmps []libdata.Clock
  dpId := c.Query("dpId")
  if dpId != "" {
    db, err := sql.Open("mssql", MssqlConnection18)
    if err != nil {
      fmt.Println("err:", err)
    }
    defer func() {
      db.Close()
      recover()
    }()
    sqlstring := "SELECT [MemberNo] FROM [SmartCard].[dbo].[Member_Info] where [OrgID]=$1"
    rows, err := db.Query(sqlstring, dpId)
    if err != nil {
      fmt.Println("Error:", "SELECT [MemberNo] FROM [SmartCard].[dbo].[Member_Info] where [OrgID]=", dpId)
      fmt.Println(err)
    } else {
      var fp_id string
      for rows.Next() {
        rows.Scan(&fp_id)
        Etmps = append(Etmps, libdata.Clock{fp_id, fp_id, ""})
      }
    }
  }
  c.JSON(200, Etmps)
}

var logger service.Logger

//應用程式轉常駐服務
func gorunServer() {
  Recv := make(chan []byte)
  flag.Parse()
  log.SetFlags(0)
  gin.SetMode(gin.ReleaseMode)
  router := gin.New()
  router.Static("/js", "./js/")

  //新增機台連線
  router.GET("/ping", func(c1 *gin.Context) {
    c, err := upgrader.Upgrade(c1.Writer, c1.Request, nil)
    if err != nil {
      fmt.Println("err:", err)
    }
    defer c.Close()
    ip := strings.Split(c1.Request.RemoteAddr, ":")
    //加入連線陣列
    mutex.Lock()
    c2 := InsertSinf(ip[0], c, FaceCon)
    mutex.Unlock()
    if c2.Id != "" {
      ReadRcv(c2)
      c.Close()
    }
  })

  //機台管理介面
  router.GET("/", web)

  //新增取得機台LIST
  router.POST("/getselects", Getselects)

  //取得部門員工LIST
  router.GET("/getempoeey", Getempoeey)

  //取得全部機台
  router.GET("/onlinemc", func(c *gin.Context) {
    Con := *FaceCon
    fmt.Println("Online:", len(Con.FaceCons))
    ts1 := getAccessControl()
    if *ts1 != nil {
      ts := *ts1
      for i, pt := range ts {
        status := "Off"
        for _, row := range Con.FaceCons {
          if row.Ip == pt.Ip && row.FaceCon != nil {
            status = "On"
            break
          }
        }
        ts[i].Status = status
      }
      c.JSON(200, ts)
    }
  })

  //機台補上傳資料[機台ID,開始時間,結束時間]
  router.POST("/importtime", func(c *gin.Context) {
    cardid := c.PostForm("cardid")
    sttime := c.PostForm("sttime")
    edtime := c.PostForm("edtime")
    db, err := sql.Open("mssql", MssqlConnection18)
    if err != nil {
      fmt.Println("err:", err)
      c.String(200, "Connection is error !!!! ")
    }
    defer func() {
      db.Close()
      recover()
    }()
    //stting Time轉Unix time
    st := sTot(sttime)
    ed := sTot(edtime)
    //1.取得機台con
    fmt.Println("WEB UPDATE....")
    tfaceCon := *FaceCon
    if cardid == "0" {
      for sno, con := range tfaceCon.FaceCons {
        if con.FaceCon != nil && con.Tsecomd != 0 {
          fmt.Println("McID:", con.Id)
          fmt.Println("Start:", sttime, "UNIX:", st)
          fmt.Println("End:", edtime, "UNIX:", ed)
          machOneUpdate1A(con, st, ed, sno)
        }
      }

    } else {
      c1, msg := tfaceCon.SearchSinf(cardid)
      if msg == "" {
        fmt.Println("McID:", cardid)
        fmt.Println("Start:", sttime, "UNIX:", st)
        fmt.Println("End:", edtime, "UNIX:", ed)
        machOneUpdate1A(c1, st, ed, 1)
      }
    }
    fmt.Println("All OK .....")
    c.String(200, "OK.....")
  })

  //資料庫->寫入刷臉機(指定員工單筆資料上傳)
  router.POST("/uppersons1", func(c *gin.Context) {
    cardid := c.PostForm("cardid")
    employee1 := c.PostForm("employee1")
    photo := c.PostForm("photo")
    db, err := sql.Open("mssql", MssqlConnection18)
    if err != nil {
      fmt.Println("err:", err)
    }
    defer func() {
      db.Close()
      recover()
    }()
    tfaceCon := *FaceCon
    c1, msg := tfaceCon.SearchSinf(cardid)
    if msg == "" && employee1 != "" {
      var fp_id, emp_fname, sex, brdate string
      sqlstring := "SELECT [MemberNo],[MemberName] ,case [Sex] when '1' then 'female' else 'male' end  [Sex]"
      sqlstring += ",[Birthday],[MemberPic] FROM [SmartCard].[dbo].[Member_Info] where [MemberNo]=$1"
      db.QueryRow(sqlstring, employee1).Scan(&fp_id, &emp_fname, &sex, &brdate, &photo)
      brdate = brdate[:10]
      fmt.Println("hpoto:", photo)
      photo = "http://192.168.20.254:8083/res/photo/0904.jpg"

      //--------------------------------
      var personrow libdata.AddPerson
      personrow.Id = rand.Intn(100)
      personrow.Method = "personnelData.savePersons"
      var persons libdata.Persons
      persons.Person.Type = 1     //人员类型,1-内部员工,2-访客, 3-黑名单；必填
      persons.Person.Code = fp_id //人员编号；必填
      //persons.Person.CredentialNo = fp_id //身份证号
      //person.GroupNames=append(person.GroupNames,"权限组一")//使用权限组时填，非必填
      persons.Person.Name = emp_fname  //姓名；必填
      persons.Person.Sex = sex         //性别；male/female；必填
      persons.Person.Birthday = brdate //生日；必填
      //person.GuestInfo =guestinfo //访客信息；Type为2时填写
      //三种下发方式只能选择一个;
      persons.Person.URL = append(persons.Person.URL, photo) //URL图片导入
      //person.Feartures=append(person.Feartures,"base64Feartures")//特征值导入
      //persons.Person.Images = append(persons.Person.Images, jpg) //base64图片导入
      /*if photo != "" {
        bytes, err := ioutil.ReadFile(photo)
        if check(err) {
          jpg := base64.StdEncoding.EncodeToString(bytes)
          persons.Person.Images = append(persons.Person.Images, jpg) //base64图片导入
        }
      }*/
      fmt.Println(sqlstring, employee1, fp_id, emp_fname, sex, brdate, photo)
      personrow.Params = append(personrow.Params, persons)
      res2B, _ := json.Marshal(personrow)
      fmt.Println(string(res2B))
      err := c1.FaceCon.WriteJSON(personrow)
      if err != nil {
        log.Println("write SendPersons:", err)
      } else {
        data1 := <-Recv
        fmt.Println(string(data1))
      }
    }
  })

  //多廠別變更圖片
  router.POST("/uppersons2", func(c *gin.Context) {
    var filelist []string
    face := c.PostForm("face")
    factory := c.PostForm("factory")                                   //廠區
    faces := strings.Split(face, ",")                                  //所有刷臉機
    myfolder := `C:\inetpub\wwwroot\SmartCard\photo2\` + factory + `\` //JPG目錄
    files, _ := ioutil.ReadDir(myfolder)
    for _, file := range files {
      if file.IsDir() {
        continue
      } else if strings.Contains(file.Name(), "jpg") {
        filelist = append(filelist, file.Name())
      }
    }
    fmt.Println(filelist)
    if len(filelist) > 0 {
      for _, faces1 := range faces { //取得所有刷臉機
        if len(faces1) > 0 {
          restpic(faces1, factory, filelist)
        }
      }
    }
  })

  //資料庫->寫入刷臉機(指定員工區間範圍多筆資料上傳)
  router.POST("/addpersons", func(c *gin.Context) {
    //------------------------------
    cardid := c.PostForm("cardid")
    dpId := c.PostForm("dpId")
    fpId1 := c.PostForm("fpId1")
    fpId2 := c.PostForm("fpId2")
    ip := strings.Split(c.Request.RemoteAddr, ":")
    db, err := sql.Open("mssql", MssqlConnection18)
    if err != nil {
      fmt.Println("err:", err)
    }
    defer func() {
      db.Close()
      recover()
    }()
    sqlwhere := "where [OrgID]='" + dpId + "' "
    if fpId1 == "" || fpId2 == "" {
      sqlwhere += "and MemberNo is not null"
    } else {
      sqlwhere += "and MemberNo between '" + fpId1 + "' and '" + fpId2 + "'"
    }
    sqlstring := "SELECT [MemberNo],[MemberName] ,case [Sex] when '1' then 'female' else 'male' end  [Sex]"
    sqlstring += ",[Birthday],[MemberPic] FROM [SmartCard].[dbo].[Member_Info]" + sqlwhere
    fmt.Println(sqlstring)
    rows, _ := db.Query(sqlstring)
    //--------------------------------
    var personrow libdata.AddPerson
    //------------------------------
    personrow.Id = rand.Intn(100)
    personrow.Method = "personnelData.savePersons"

    var persons libdata.Persons
    var person libdata.Person
    for rows.Next() {
      person = libdata.Person{}
      var fp_id, emp_fname, sex, birth_date, photo string
      rows.Scan(&fp_id, &emp_fname, &sex, &birth_date, &photo)
      person.Type = 1             //人员类型,1-内部员工,2-访客, 3-黑名单；必填
      person.Code = fp_id         //人员编号；必填
      person.CredentialNo = fp_id //身份证号
      //person.GroupNames=append(person.GroupNames,"权限组一")//使用权限组时填，非必填
      person.Name = emp_fname      //姓名；必填
      person.Sex = sex             //性别；male/female；必填
      person.Birthday = birth_date //生日；必填
      //person.GuestInfo =guestinfo //访客信息；Type为2时填写
      //三种下发方式只能选择一个;
      person.URL = append(person.URL, "http://"+ip[0]+":8083/"+photo) //URL图片导入
      //person.Feartures=append(person.Feartures,"base64Feartures")//特征值导入
      //person.Images = append(person.Images, base64.StdEncoding.EncodeToString(photo)) //base64图片导入
      //person.Cards=cards//卡号字段；非必填
      persons.Person = person
      personrow.Params = append(personrow.Params, persons)
    }
    if len(personrow.Params) > 0 {
      //res2B, _ := json.Marshal(personrow)
      //fmt.Println(string(res2B))
      //1.取得機台con
      //fmt.Println(cardid, "暫停寫入刷臉機")

      tfaceCon := *FaceCon
      c1, msg := tfaceCon.SearchSinf(cardid)
      if msg == "" {
        err := c1.FaceCon.WriteJSON(personrow)
        if err != nil {
          log.Println("write SendPersons:", err)
        } else {
          data1 := <-Recv
          fmt.Println(string(data1))
        }
      }
    }
  })

  //刷臉機->寫入資料庫
  router.POST("/dlpersons", func(c *gin.Context) {
    //------------------------------
    cardid := c.PostForm("cardid")
    db, err := sql.Open("mssql", MssqlConnection18)
    if err != nil {
      fmt.Println("err:", err)
    }
    defer func() {
      db.Close()
      recover()
    }()
    tfaceCon := *FaceCon
    c1, msg := tfaceCon.SearchSinf(cardid)
    if msg == "" {
      //2.SEARCH人員LIST
      err := c1.FaceCon.WriteJSON(libdata.SendPersons(0))
      if err != nil {
        log.Println("Get Persons list:", err)
      }
      //3.取得所有人員名單
      var record = libdata.GetAllPersons{}
      data := <-Recv
      err = json.Unmarshal(data, &record)
      if err != nil {
        fmt.Println("ERROR:", err)
      }
      for _, row := range record.Params.Persons {
        sqlstring := "INSERT INTO [livil_person] ([ep_id],[mh_id]) VALUES($1,$2)"
        db.Exec(sqlstring, cardid, row.Code)
      }
    }
  })

  router.POST("/setconnection", func(c *gin.Context) {
    src_ip := c.PostForm("src_ip")
    src_us := c.PostForm("src_us")
    src_pw := c.PostForm("src_pw")
    if src_ip != "" && src_us != "" && src_pw != "" {
      constring := "server=" + src_ip + ";user id=" + src_us + ";password=" + src_pw + ";database=SmartCard"
      //winsetregisty(constring)
      //c.JSON(200, constring)
      fmt.Println(constring)
      ts := Ecodecs(constring)
      fmt.Println(constring)
      resetconfig(ts)
      c.JSON(200, ts)
    }
  })

  //全部補資料
  router.POST("/unconnection", func(c *gin.Context) {
    src_string := c.PostForm("src_string")
    ts := Dcodecs(src_string)
    c.JSON(200, ts)
  })
  //fmt.Println("Ws 1234")
  //router.Run(":1234")
  server1 := &http.Server{
    Addr:    ":1234",
    Handler: router,
  }
  ch := make(chan int)
  if ttime := strings.Split(Restart, ":"); Restart != "" && len(ttime) == 2 {
    fmt.Println("Restart server Hour:", ttime[0], "Minute:", ttime[1])
    go func() {
      hs, _ := strconv.Atoi(ttime[0])
      ms, _ := strconv.Atoi(ttime[1])
      T1 := time.NewTicker(1 * time.Minute)
      for {
        tt := <-T1.C
        if tt.Minute() == ms && tt.Hour() == hs {
          for {
            err := server1.Close()
            if err == nil {
              ch <- 1
              break
            }
          }
          log.Println("server1.Close Hour:", tt.Hour(), "Minute:", tt.Minute())
          time.Sleep(10 * time.Second)
          return
        }
      }
    }()
  }
  /*select {
    case <-ch:
      return
    }*/
  server1.ListenAndServe()
}

//自動轉檔程式
func autoinit(FaceCon *libdata.Sinfs) {
  time.Sleep(20 * time.Second)
  thisTime := time.Now()

  arear := strings.Split(ZoneTime, ",")
  ttime := strings.Split(arear[1], ":")
  h1, _ := strconv.Atoi(ttime[0])
  h2, _ := strconv.Atoi(ttime[1])
  utc := time.FixedZone(arear[1], (h1*60*60)+(h2*60))

  if eveTimes > 0 {
    fmt.Println("連續定時:", eveTimes)
  } else if len(TimesGos) > 0 {
    fmt.Println("排程:", TimesGos)
  }
  fmt.Println("Shedule start:", thisTime.Year(), "-", int(thisTime.Month()), "-", thisTime.Day(), " ", thisTime.Hour(), ":", thisTime.Minute())

  T1 := time.NewTicker(1 * time.Minute)
  for {
    tt := <-T1.C
    if eveTimes > 0 {
      if (tt.Minute() % eveTimes) == 0 {
        m, _ := time.ParseDuration("-" + BefoMinutes + "m")
        thisTime1 := tt.Add(m)
        thisTime1 = time.Date(thisTime1.Year(), thisTime1.Month(), thisTime1.Day(), thisTime1.Hour(), thisTime1.Minute(), 0, 0, utc)
        tt = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), 0, 0, utc)
        fmt.Println("Auto start:", thisTime1.Format("2006.01.02 15:04:05"), " To ", tt.Format("2006.01.02 15:04:05"))
        if len((*FaceCon).FaceCons) > 0 {
          st2 := thisTime1.Unix()
          ed2 := tt.Unix()
          for sno, con := range (*FaceCon).FaceCons {
            if con.FaceCon != nil {
              machOneUpdate1A(con, st2, ed2, sno)
            }
          }
          fmt.Println(" IS All OK .....")
        }
      }
    } else if len(TimesGos) > 0 {
      for index, TGo := range TimesGos {
        if TGo.Hours == tt.Hour() && TGo.Minutes == tt.Minute() && (TimesGos[index].Days != tt.Day()) {
          fmt.Println("Schedule start:", tt.Format("2006.01.02 15:04:05"))
          h, _ := time.ParseDuration(OnceHours[index])
          tt2 := tt.Add(h)
          tt2 = time.Date(tt2.Year(), tt2.Month(), tt2.Day(), tt2.Hour(), tt2.Minute(), 0, 0, utc)
          tt = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), 0, 0, utc)
          fmt.Println(index, "-", tt2.Format("2006.01.02 15:04:05"), " To ", tt.Format("2006.01.02 15:04:05"))
          msg := "Schedule start:" + tt2.Format("2006.01.02 15:04:05") + " To " + tt.Format("2006.01.02 15:04:05")
          WriteLog(msg)
          tfaceCon := *FaceCon
          if len(tfaceCon.FaceCons) > 0 {
            st2 := tt2.Unix()
            ed2 := tt.Unix()
            for sno, con := range tfaceCon.FaceCons {
              if con.FaceCon != nil {
                machOneUpdate1A(con, st2, ed2, sno)
              }
            }
            TimesGos[index].Days = tt.Day()
            fmt.Println(TGo, index, " IS All OK .....")
          } else {
            WriteLog("No Any Connetions !!!!")
          }
        }
      }
    }
  }
}

func arragetest() {
  arear := strings.Split(ZoneTime, ",")
  ttime := strings.Split(arear[1], ":")
  h1, _ := strconv.Atoi(ttime[0])
  h2, _ := strconv.Atoi(ttime[1])
  utc := time.FixedZone(arear[1], (h1*60*60)+(h2*60))
  if len(TimesGos) > 0 {
    tt := time.Now()
    for index, TGo := range TimesGos {
      fmt.Println("定時:", TGo)
      fmt.Println("Schedule start:", tt.Format("2006.01.02 15:04:05"))
      h, _ := time.ParseDuration(OnceHours[index])
      tt2 := tt.Add(h)
      fmt.Println("現在時間:", tt, " + ", OnceHours[index], tt2)
      tt2 = time.Date(tt2.Year(), tt2.Month(), tt2.Day(), tt2.Hour(), tt2.Minute(), 0, 0, utc)
      tt = time.Date(tt.Year(), tt.Month(), tt.Day(), tt.Hour(), tt.Minute(), 0, 0, utc)
      fmt.Println(index, "-", tt2.Format("2006.01.02 15:04:05"), " To ", tt.Format("2006.01.02 15:04:05"))
    }
  }
}

func main() {
  loadconfig() //載入設定檔
  /*MssqlConnection18 = Dcodecs(MssqlConnection18)
    fmt.Println("原文：", MssqlConnection18)
    orig := MssqlConnection18
      fmt.Println("原文：", orig)
      encryptCode := AesEncrypt(orig)
      fmt.Println("密文：", encryptCode)
      decryptCode := AesDecrypt(encryptCode)
      fmt.Println("解密結果：", decryptCode)*/
  //MssqlConnection18 = winregisty()
  var faceCon = libdata.Sinfs{}
  ts1 := getAccessControl() //取得資料庫註冊刷臉機
  if *ts1 != nil {
    for _, pt := range *ts1 {
      stmp := libdata.Sinf{pt.Id, pt.Ip, pt.Name, nil, 0, nil}
      faceCon.FaceCons = append(faceCon.FaceCons, stmp)
    }
    FaceCon = &faceCon //建立連線檔
    fmt.Println("機台總數:", len(faceCon.FaceCons))
    //自動補資料
    if len(faceCon.FaceCons) > 0 {
      go autoinit(FaceCon)
    }
    //自動重新連線
    //for {
    //  fmt.Println("Start---------")
    gorunServer()
    fmt.Println("Close Restart---------")
    //}
  }
}

/*
func deletefacedata(c1 libdata.Sinf) {
  st := sTot("2022-06-01")
  ed := sTot("2022-06-10")
  mdata := libdata.SelectData{}
  mdata.Id = rand.Intn(1000)
  mdata.Method = "accessRecord.remove"
  mdata.Params.Condition.PersonCode = "90917"
  mdata.Params.Condition.StartTime = st
  mdata.Params.Condition.EndTime = ed
  mdata.Params.Condition.AccessType = "Face"
  mdata.Params.Condition.Offset = i * 1000
  mdata.Params.Condition.Limit = 1000
  err := c1.FaceCon.WriteJSON(mdata)
  if err != nil {
    log.Println("delete record:", err)
    WriteLog("delete record:" + err.Error())
    return
  } else {
    var record1 = libdata.GetData{}
    data1 := <-c1.Recv
    err = json.Unmarshal(data1, &record1)
    if err != nil {
      log.Println("delete 2 record index:", sno, err)
      msg += "delete record Err:" + err.Error() + "\r\n"
    }
  }
}
*/
