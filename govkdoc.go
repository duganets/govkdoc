package govkdoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
)

type VkConn struct {
	accToken string
}

type vkDoc struct {
	ownerId uint
	did     uint
}

func NewVkConn(accToken string) *VkConn {
	return &VkConn{accToken}
}

func (vc VkConn) WallShareFileAsDoc(text string, filePath string) error {
	var uploadServerUrl string

	log.Printf("wallShare text=[%s] file=[%s] token=[%s]", text, filePath, vc.accToken)

	uploadServerUrl, upSrvErr := vc.getUploadServer()
	if upSrvErr != nil {
		log.Print("getUploadServer failed: ", upSrvErr)
		return upSrvErr
	}
	log.Printf("uploadServerUrl=%s", uploadServerUrl)

	upFile, upErr := vc.upload(uploadServerUrl, filePath)
	if upErr != nil {
		log.Print("upload failed: ", upErr)
		return upErr
	}
	log.Printf("uploaded file=%s", upFile)

	doc, saveErr := vc.docSave(upFile)

	if saveErr != nil {
		log.Print("docSave failed: ", saveErr)
		return saveErr
	}
	log.Print("saved doc=%q", doc)

	postErr := vc.wallPost(text, doc)

	if postErr != nil {
		log.Print("wallPost failed: ", postErr)
		return postErr
	}
	log.Print("wallShareFile ok")
	return nil
}

func (vc VkConn) wallPost(text string, doc *vkDoc) error {
	atts := fmt.Sprintf("%s%d_%d", "doc", doc.ownerId, doc.did)
	uploadUrl := "https://api.vk.com/method/wall.post?friends_only=0&from_group=0&access_token=" +
		vc.accToken + "&message=" + text + "&attachments=" + atts

	_, errWall := http.Get(uploadUrl)
	if errWall != nil {
		log.Print(uploadUrl)
		return errWall
	}
	return nil
	/*rawRespWall, errRespWall := ioutil.ReadAll(respWall.Body)
	if errRespWall != nil {
		log.Print("error: ", errRespWall)
		return
	}

	log.Print("wall.post resp=", string(rawRespWall))*/
}

func (vc VkConn) docSave(docFile string) (*vkDoc, error) {
	var vkRespMul map[string][]map[string]interface{}

	respSave, errSave := http.Get("https://api.vk.com/method/docs.save?access_token=" + vc.accToken + "&file=" + docFile)
	if errSave != nil {
		log.Print(respSave)
		return nil, errSave
	}
	rawSaveBody, readSaveBodyErr := ioutil.ReadAll(respSave.Body)
	if readSaveBodyErr != nil {
		log.Print(respSave)
		return nil, readSaveBodyErr
	}

	errSaveRespUnm := json.Unmarshal(rawSaveBody, &vkRespMul)
	if errSaveRespUnm != nil {
		log.Print(string(rawSaveBody))
		return nil, errSaveRespUnm
	}

	_, rok := vkRespMul["response"]
	if !rok {
		return nil, errors.New("jsonObj->response not found")
	}
	if len(vkRespMul["response"]) == 0 {
		return nil, errors.New("jsonObj->response[] has no items")
	}

	ow, owExs := vkRespMul["response"][0]["owner_id"]
	if !owExs {
		return nil, errors.New("jsonObj->response[0]->owner_id not found")
	}
	owF, owParseOk := ow.(float64)
	if !owParseOk {
		return nil, errors.New("can't parse owner_id, float64 expected")
	}

	dId, didExs := vkRespMul["response"][0]["did"]
	if !didExs {
		return nil, errors.New("jsonObj->response[0]->did not found")
	}

	dIdF, dParseOk := dId.(float64)
	if !dParseOk {
		return nil, errors.New("can't parse did, float64 expected")
	}

	return &vkDoc{uint(owF), uint(dIdF)}, nil
}

func (vc VkConn) upload(uploadUrl string, filePath string) (string, error) {
	var b bytes.Buffer
	var vkRespSave map[string]string

	uploadFile, _ := ioutil.ReadFile(filePath)

	w := multipart.NewWriter(&b)
	fw, errFF := w.CreateFormFile("file", "upload.gif") // TODO define filename
	if errFF != nil {
		return "", errFF
	}
	_, errCopy := io.Copy(fw, bytes.NewReader(uploadFile))
	if errCopy != nil {
		return "", errCopy
	}
	upReq, upReqErr := http.NewRequest("POST", uploadUrl, &b)
	if upReqErr != nil {
		return "", upReqErr
	}
	upReq.Header.Set("Content-Type", w.FormDataContentType())
	w.Close()

	client := &http.Client{}

	upReq.ContentLength = -1
	//	rbs, _ := httputil.DumpRequest(upReq, false)

	upResp, upErr := client.Do(upReq)
	if upErr != nil {
		log.Print(upResp)
		return "", upErr
	}
	rawUpRespBody, errReadUpBody := ioutil.ReadAll(upResp.Body)
	if errReadUpBody != nil {
		log.Print(upResp)
		return "", errReadUpBody
	}

	errUpRespUnm := json.Unmarshal(rawUpRespBody, &vkRespSave)
	if errUpRespUnm != nil {
		log.Print(string(rawUpRespBody))
		return "", errUpRespUnm
	}

	uploadedFile, isExs := vkRespSave["file"]
	if !isExs {
		return "", errors.New("jsonObj->file not found")
	}
	return uploadedFile, nil
}

func (vc VkConn) getUploadServer() (string, error) {
	var vkResp map[string]map[string]string

	resp, reqErr := http.Get("https://api.vk.com/method/docs.getWallUploadServer?access_token=" + vc.accToken)
	if reqErr != nil {
		log.Print(resp)
		return "", reqErr
	}
	rawBody, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Print(resp)
		return "", readErr
	}
	errUnm := json.Unmarshal(rawBody, &vkResp)
	if errUnm != nil {
		log.Print(resp)
		return "", errUnm
	}
	_, rok := vkResp["response"]
	if !rok {
		return "", errors.New("jsonObj->response not found")
	}
	upUrl, uuok := vkResp["response"]["upload_url"]
	if !uuok {
		return "", errors.New("jsonObj->response->upload_url not found")
	}
	return upUrl, nil
}
