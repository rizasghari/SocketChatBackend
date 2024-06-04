package servers

type HttpServer struct {
}

func NewHttpServer() *HttpServer {
	return &HttpServer{}
}

func (hs *HttpServer) Run() error {
	return nil
}
