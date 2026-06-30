package mailmgmt

type ListedAddress struct {
	Raw            string
	Email          string
	MaxQuota       string
	CurrentQuota   string
	PercentageFull int
	Aliases        []string
}
