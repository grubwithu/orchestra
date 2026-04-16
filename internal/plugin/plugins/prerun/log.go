package prerun

type Log struct {
	Cov int `json:"cov"`
}

func (pd *PrerunData) GetLog() any {
	return &Log{
		Cov: pd.Cov,
	}
}
