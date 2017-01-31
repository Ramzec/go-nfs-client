package nfs

// MOUNT
// RFC 1813 Section 5.0

import (
	"fmt"

	"github.com/davecheney/nfs/rpc"
	"github.com/davecheney/nfs/xdr"
)

const (
	MOUNT_PROG = 100005
	MOUNT_VERS = 3

	MOUNTPROC3_NULL   = 0
	MOUNTPROC3_MNT    = 1
	MOUNTPROC3_UMNT   = 3
	MOUNTPROC3_EXPORT = 5

	MNT3_OK             = 0     // no error
	MNT3ERR_PERM        = 1     // Not owner
	MNT3ERR_NOENT       = 2     // No such file or directory
	MNT3ERR_IO          = 5     // I/O error
	MNT3ERR_ACCES       = 13    // Permission denied
	MNT3ERR_NOTDIR      = 20    // Not a directory
	MNT3ERR_INVAL       = 22    // Invalid argument
	MNT3ERR_NAMETOOLONG = 63    // Filename too long
	MNT3ERR_NOTSUPP     = 10004 // Operation not supported
	MNT3ERR_SERVERFAULT = 10006 // A failure on the server
)

type Export struct {
	Dir    string
	Groups []Group
}

type Group struct {
	Name string
}

type Mount struct {
	*rpc.Client
	dirPath string
	Addr    string
}

func (m *Mount) Unmount() error {
	type umount struct {
		rpc.Header
		dirpath string
	}

	_, err := m.Call(&umount{
		rpc.Header{
			Rpcvers: 2,
			Prog:    MOUNT_PROG,
			Vers:    MOUNT_VERS,
			Proc:    MOUNTPROC3_UMNT,
			Cred:    rpc.AUTH_NULL,
			Verf:    rpc.AUTH_NULL,
		},
		m.dirPath,
	})
	if err != nil {
		return err
	}

	return nil
}

func (m *Mount) Mount(dirpath string, auth rpc.Auth) (*Volume, error) {
	type mount struct {
		rpc.Header
		Dirpath string
	}

	buf, err := m.Call(&mount{
		rpc.Header{
			Rpcvers: 2,
			Prog:    MOUNT_PROG,
			Vers:    MOUNT_VERS,
			Proc:    MOUNTPROC3_MNT,
			Cred:    auth,
			Verf:    rpc.AUTH_NULL,
		},
		dirpath,
	})
	if err != nil {
		return nil, err
	}

	mountstat3, buf := xdr.Uint32(buf)
	switch mountstat3 {
	case MNT3_OK:
		fh, buf := xdr.Opaque(buf)
		_, buf = xdr.Uint32List(buf)

		m.dirPath = dirpath

		vol, err := NewTarget("tcp", m.Addr, auth, fh, dirpath)
		if err != nil {
			return nil, err
		}

		return vol, nil

	case MNT3ERR_PERM:
		return nil, &Error{"MNT3ERR_PERM"}
	case MNT3ERR_NOENT:
		return nil, &Error{"MNT3ERR_NOENT"}
	case MNT3ERR_IO:
		return nil, &Error{"MNT3ERR_IO"}
	case MNT3ERR_ACCES:
		return nil, &Error{"MNT3ERR_ACCES"}
	case MNT3ERR_NOTDIR:
		return nil, &Error{"MNT3ERR_NOTDIR"}
	case MNT3ERR_NAMETOOLONG:
		return nil, &Error{"MNT3ERR_NAMETOOLONG"}
	}
	return nil, fmt.Errorf("unknown mount stat: %d", mountstat3)
}

// TODO(dfc) unfinished
func (m *Mount) Exports() ([]Export, error) {
	type export struct {
		rpc.Header
	}
	msg := &export{
		rpc.Header{
			Rpcvers: 2,
			Prog:    MOUNT_PROG,
			Vers:    MOUNT_VERS,
			Proc:    MOUNTPROC3_EXPORT,
			Cred:    rpc.AUTH_NULL,
			Verf:    rpc.AUTH_NULL,
		},
	}
	_, err := m.Call(msg)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func DialMount(nt, addr string) (*Mount, error) {
	// get MOUNT port
	m := rpc.Mapping{
		Prog: MOUNT_PROG,
		Vers: MOUNT_VERS,
		Prot: rpc.IPPROTO_TCP,
		Port: 0,
	}

	client, err := DialService(nt, addr, m)
	if err != nil {
		return nil, err
	}

	return &Mount{
		Client: client,
		Addr:   addr,
	}, nil
}
