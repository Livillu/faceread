package libdata

import (
  "github.com/gorilla/websocket"
)

//照片查詢----------------------
//建立查詢物件
type CreatePicture struct {
  Id     int    `json:"id"`
  Method string `json:"method"`
}

type CreateResult struct {
  Id     int `json:"id"`
  Result int `json:"result"`
}

//查詢照片
type SearchPicture struct {
  Id     int        `json:"id"`
  Object int        `json:"object"`
  Method string     `json:"method"`
  Params InfPicture `json:"params"`
}

type InfPicture struct {
  CertificateType string
  ID              string
}

//查詢結果
type SearchResult struct {
  Id             int    `json:"id"`
  Params         string `json:"params"`
  FaceImageInfos []InfPictures
}

type InfPictures struct {
  FaceToken string
  PersonID  int
  GroupID   int
  State     int
  AddTime   string
}

//刪除查詢物件
type DelPicture struct {
  Id     int    `json:"id"`
  Object int    `json:"object"`
  Method string `json:"method"`
}
type DelResult struct {
  Id     int `json:"id"`
  Result int `json:"result"`
}

//---------------------------------------------------------------
//查詢員工編號
type CPerson struct {
  Type     int
  CodeLike string
  NameLike string
  Limit    int
  Offset   int
}

type CPersons struct {
  Condition CPerson
}

type SelectPersons struct {
  Id     int      `json:"id"`
  Method string   `json:"method"`
  Params CPersons `json:"params"`
}

//取得員工資料
type GetPerson struct {
  //AccessTimes int
  //Birthday   string
  Code string
  //Custom     string
  //GroupName  string
  //HealthCode int
  //Name       string
  //Sex        string
  //Status     int
  //Type       int
}

type GetPersons struct {
  Persons []GetPerson
}

type GetAllPersons struct {
  Id     int        `json:"id"`
  Method string     `json:"method"`
  Params GetPersons `json:"params"`
}

//-----------------------------------------------------------------

//查詢刷卡紀錄
type Condition struct {
  PersonCode string `json:"PersonCode,omitempty"`
  AccessType string
  StartTime  int64
  EndTime    int64
  Offset     int
  Limit      int
}

type Conditions struct {
  Condition Condition
}

type SelectData struct {
  Id     int        `json:"id"`
  Method string     `json:"method"`
  Params Conditions `json:"params"`
}

//查詢刷卡紀錄

//-----------------------------------------------------------------

//取得刷卡紀錄
type Records struct {
  Records []Record
}

type Record struct {
  //AccessType     string
  //CardNo         string
  //CardType       int
  //CredentialNo   string
  //CredentialType string
  //Detail         string
  //ID             int
  //IdCard         string
  //Mask           int
  Pass       bool
  PersonCode string
  //PersonName     string
  //PersonType     int
  //SearchScore    float32
  Time int64
}

type GetData struct {
  Id     int     `json:"id"`
  Method string  `json:"method"`
  Params Records `json:"params"`
}

//websocket[]
type Sinfs struct {
  FaceCons []Sinf
}
type Sinf struct {
  Id      string
  Ip      string
  Name    string
  FaceCon *websocket.Conn
  Tsecomd int64
  Recv    chan []byte
}

type Clock struct {
  Id   string
  Name string
  Ip   string
}

//資料庫->寫入刷臉機
type AccessTime struct {
  From int64 `json:"from"`
  To   int64 `json:"to"`
}
type GuestInfo struct {
  Corp       string
  Phone      string
  CarLicense string
  Partner    int
  Host       string
  AccessTime []AccessTime
}
type Memo struct {
  Entrance string
}

type Card struct {
  ID           string
  Type         int
  Validity     []string
  ValidityTime []string
  Memo         Memo
}
type Person struct {
  Type            int
  Code            string
  CertificateType string   `json:"CertificateType,omitempty"`
  CredentialNo    string   `json:"CredentialNo,omitempty"`
  GroupNames      []string `json:"GroupNames,omitempty"`
  Name            string
  Sex             string
  Birthday        string
  //GuestInfo    GuestInfo
  URL       []string `json:"URL,omitempty"`
  Feartures []string `json:"Feartures,omitempty"`
  Images    []string `json:"Images,omitempty"`
  //Cards     []Card   `json:"Cards,omitempty"`
}
type Persons struct {
  Person Person
}
type AddPerson struct {
  Id     int       `json:"id"`
  Method string    `json:"method"`
  Params []Persons `json:"params"`
}

//-----------圖片處理--------------
type InsertPerson struct {
  Id     int    `json:"id"`
  Method string `json:"method"`
  Params Person `json:"params"`
}

type CreateImg struct {
  Id     int    `json:"id"`
  Method string `json:"method"`
}

type PImag struct {
  Amount  int
  Lengths []int
}

type UpPerson struct {
  GroupID    int
  PersonInfo Person
  ImageInfo  PImag
}

type UpdateAddFace struct {
  Id     int      `json:"id"`
  Method string   `json:"method"`
  Params UpPerson `json:"params"`
}

//--------------------
//機台
type AccessControl struct {
  Id     string
  Ip     string
  Name   string
  Status string
}
