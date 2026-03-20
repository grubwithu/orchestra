package webcore

func (s *Server) setupRoutes() {
	// POST
	s.Router.POST("/reportCorpus", s.handleReportCorpus)
	s.Router.POST("/log", s.handleLog)

	// GET
	s.Router.GET("/peekResult", s.handlePeekResult)
	s.Router.GET("/ready", s.handleReady)
}
