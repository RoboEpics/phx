package pei

type peiRequest interface {
	script() string
	env() map[string]string
}

// Pack directory and project results to a X.tar.gz file.
type Pack struct {
	// Pack into $EGG_PACK env variable
	EggPack string
}

func (p Pack) script() string { return "pack" }
func (p Pack) env() map[string]string {
	return map[string]string{
		"EGG_PACK": p.EggPack,
	}
}

// TAR project in order to send to remote peer.
type TAR struct {
	// Tar current directory into $TAR_FILE
	TarFile string
}

func (t TAR) script() string { return "tar" }
func (t TAR) env() map[string]string {
	return map[string]string{
		"TAR_FILE": t.TarFile,
	}
}

// Unpack packed result.
type Unpack struct {
	// Unpack $EGG_PACK result file
	EggPack string
}

func (u Unpack) script() string { return "unpack" }
func (u Unpack) env() map[string]string {
	return map[string]string{
		"EGG_PACK": u.EggPack,
	}
}
