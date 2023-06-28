package libdata

import (
  "log"
  "math/rand"
  "time"
  //"github.com/gorilla/websocket"
)

//查詢員工編號
func SendPersons(index int) SelectPersons {
  var data = SelectPersons{}
  data.Id = rand.Intn(100)
  data.Method = "personManager.getPersons"
  data.Params.Condition.Type = 1
  data.Params.Condition.CodeLike = ""
  data.Params.Condition.NameLike = ""
  data.Params.Condition.Limit = 1000
  data.Params.Condition.Offset = index
  return data
}

//get current websocket
func (Tc Sinfs) SearchSinf(id string) (Sinf, string) {
  msg := "No Record"
  var tmp = Sinf{}
  for _, row := range Tc.FaceCons {
    if id == row.Id && row.Tsecomd != 0 {
      tmp = row
      msg = ""
      break
    }
  }
  return tmp, msg
}
func (Tc Sinfs) RemoveSinf(ip string) Sinfs {
  total := len(Tc.FaceCons)
  nowRow := 0
  if total > 0 {
    for index, row := range Tc.FaceCons {
      nowRow = index
      if ip == row.Ip {
        err := row.FaceCon.Close()
        if err != nil {
          log.Println(time.Now().Format("2006.01.02 15:04:05"), "remove Ip:", row.Ip, "FaceCon.Close Error!!!", err)
        } else {
          row.Recv = make(chan []byte)
          row.Tsecomd = 0
          log.Println("remove>> Ip:", row.Ip, time.Now().Format("2006.01.02 15:04:05"))
        }
        break
      }
    }
    log.Println("remove>> FaceCons.Tsecomd:", Tc.FaceCons[nowRow].Tsecomd)
  }
  return Tc
}

func (c InsertPerson) InsertExemple() {
  c.Id = 590
  c.Method = "personManager.insertPerson"
  c.Params.Type = 1
  c.Params.CertificateType = "IC"
  c.Params.Code = "32647"
  c.Params.Name = "Kyi Kyi Win"
  c.Params.Sex = "female"
  c.Params.Birthday = "2017-10-23"
  c.Params.GroupNames = []string{"test"}
}

func (c UpdateAddFace) InsertExemple() {
  c.Id = 592
  c.Method = "faceInfoUpdate.addFace"
  c.Params.GroupID = 1
  c.Params.PersonInfo.Name = "Kyi Kyi Win"
  c.Params.PersonInfo.Birthday = "2017-10-23"
  c.Params.PersonInfo.Sex = "female"
  c.Params.PersonInfo.CertificateType = "IC"
  //c.Params.PersonInfo.ID = "32647"
  c.Params.ImageInfo.Amount = 1
  c.Params.ImageInfo.Lengths = append(c.Params.ImageInfo.Lengths, 91123)
}
