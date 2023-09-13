package models

type Region string

const (
	RegionAWSOregon  = Region("aws-us-west-2")
	RegionAliBeijing = Region("ali-cn-beijing")
)

var (
	DefaultRegion = RegionAWSOregon
)
