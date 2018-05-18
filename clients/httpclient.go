package clients

type HttpClient struct {
}

func NewHttpClient() HttpClient {
	return HttpClient{}
}

//func (c HttpClient) Send(nodeId int, tx models.Tx, stage string) {
//	b := jsonize(tx)
//	go http.Post(url(nodeId, stage), "application/json", b)
//}
//
//func url(nodeId int, stage string) string {
//	return fmt.Sprintf("http://localhost:808%d", nodeId) + "/consensus/" + stage
//}

//func jsonize(tx models.Tx) *bytes.Buffer {
//	b := new(bytes.Buffer)
//	json.NewEncoder(b).Encode(tx)
//	return b
//}
