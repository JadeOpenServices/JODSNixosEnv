package oddcvalidation

type WritebackMode string

const (
	WritebackLocalOnly WritebackMode = "local-only"
	WritebackUpstream  WritebackMode = "upstream"
)

func DecideWriteback(authority Authority) WritebackMode {
	if authority.Allowed {
		return WritebackUpstream
	}
	return WritebackLocalOnly
}
