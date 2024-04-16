package p

type benchmark struct {
	BeginningTime int64 `json:"beginning_time,omitempty"`
	EndingTime    int64 `json:"ending_time,omitempty"`
	DeltaETandBT  int64 `json:"delta_et_bt,omitempty"`
}
