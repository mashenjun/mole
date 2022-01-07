package consts

const (
	HeatMapTypeReadKeys   = "read_keys"
	HeatMapTypeReadBytes  = "read_bytes"
	HeatMapTypeWriteKeys  = "written_keys"
	HeadMapTypeWriteBytes = "written_bytes"
)

const (
	MetricStep = 15 // use 15s step, also 15 seconds is the minimal step
)

const (
	ConvertorProcessLastLevelRatio = "last_level_ratio"
	ConvertorProcessFillGap        = "fill_gap"
	ConvertorProcessDropGap        = "drop_gap"
)

const (
	MetaYamlName = "meta.yaml"
)

const (
	PromPathRangeQuery   = "/api/v1/query_range"
	PromPathInstantQuery = "/api/v1/query"
	PromPathLabelList    = "/api/v1/label/__name__/values"

	PromExprInstanceCnt = `count(probe_success{tidb_cluster="%s", group="%s"})`
)

const (
	VMSelectPromPrefix = `/select/%d/prometheus`
)
