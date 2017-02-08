package dagcmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	cmds "github.com/ipfs/go-ipfs/commands"
	path "github.com/ipfs/go-ipfs/path"

	eth "github.com/ipfs/go-ipld-eth"
	zec "github.com/ipfs/go-ipld-zcash"
	btc "gx/ipfs/QmSDHtBWfSSQABtYW7fjnujWkLpqGuvHzGV3CUj9fpXitQ/go-ipld-btc"
	cid "gx/ipfs/QmV5gPoRsjN1Gid3LMdNZTyfCtP2DsvqEbMAmz82RmmiGk/go-cid"
	ipldcbor "gx/ipfs/QmW59q2Xq33S7LLnjzUUqbVoYyWd3TP4iMedQF8MKk2U3e/go-ipld-cbor"
	node "gx/ipfs/QmYDscK7dmdo2GZ9aumS8s5auUUAH5mR1jvj5pYhWusfK7/go-ipld-node"
)

var DagCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Interact with ipld dag objects.",
		ShortDescription: `
'ipfs dag' is used for creating and manipulating dag objects.

This subcommand is currently an experimental feature, but it is intended
to deprecate and replace the existing 'ipfs object' command moving forward.
		`,
	},
	Subcommands: map[string]*cmds.Command{
		"put":     DagPutCmd,
		"get":     DagGetCmd,
		"resolve": DagResolveCmd,
		"tree":    DagTreeCmd,
	},
}

type OutputObject struct {
	Cid *cid.Cid
}

var DagPutCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Add a dag node to ipfs.",
		ShortDescription: `
'ipfs dag put' accepts input from a file or stdin and parses it
into an object of the specified format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.FileArg("object data", true, false, "The object to put").EnableStdin(),
	},
	Options: []cmds.Option{
		cmds.StringOption("format", "f", "Format that the object will be added as.").Default("cbor"),
		cmds.StringOption("input-enc", "Format that the input object will be.").Default("json"),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		fi, err := req.Files().NextFile()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		ienc, _, _ := req.Option("input-enc").String()
		format, _, _ := req.Option("format").String()

		switch ienc {
		case "json":
			nd, err := convertJsonToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			c, err := n.DAG.Add(nd)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			res.SetOutput(&OutputObject{Cid: c})
			return
		case "hex":
			nds, err := convertHexToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			blkc, err := n.DAG.Add(nds[0])
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			if len(nds) > 1 {
				for _, nd := range nds[1:] {
					_, err := n.DAG.Add(nd)
					if err != nil {
						res.SetError(err, cmds.ErrNormal)
						return
					}
				}
			}

			res.SetOutput(&OutputObject{Cid: blkc})
		case "raw":
			nds, err := convertRawToType(fi, format)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			blkc, err := n.DAG.Add(nds[0])
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
			}

			if len(nds) > 1 {
				for _, nd := range nds[1:] {
					_, err := n.DAG.Add(nd)
					if err != nil {
						res.SetError(err, cmds.ErrNormal)
						return
					}
				}
			}

			res.SetOutput(&OutputObject{Cid: blkc})
		default:
			res.SetError(fmt.Errorf("unrecognized input encoding: %s", ienc), cmds.ErrNormal)
			return
		}
	},
	Type: OutputObject{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			oobj, ok := res.Output().(*OutputObject)
			if !ok {
				return nil, fmt.Errorf("expected a different object in marshaler")
			}

			return strings.NewReader(oobj.Cid.String()), nil
		},
	},
}

var DagGetCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Get a dag node from ipfs.",
		ShortDescription: `
'ipfs dag get' fetches a dag node from ipfs and prints it out in the specifed format.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("ref", true, false, "The object to get").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, rem, err := n.Resolver.ResolveToLastNode(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		var out interface{} = obj
		if len(rem) > 0 {
			final, _, err := obj.Resolve(rem)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			out = final
		}

		res.SetOutput(out)
	},
}

var DagResolveCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Resolve a path through an ipld node.",
		ShortDescription: `
'ipfs dag resolve' fetches a dag node from ipfs and resolves a path through it.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("path", true, false, "The path to resolve."),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, err := n.Resolver.ResolvePath(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(obj)
	},
}

var DagTreeCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Enumerate paths within a given ipld node.",
		ShortDescription: `
'ipfs dag tree' lists paths through the given ipld node
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("object", true, false, "The object to list paths of."),
	},
	Options: []cmds.Option{
		cmds.IntOption("depth", "d", "depth of listing (-1 for no limit)").Default(-1),
		cmds.StringOption("path", "path within the given object to list from"),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		p, err := path.ParsePath(req.Arguments()[0])
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		obj, err := n.Resolver.ResolvePath(req.Context(), p)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		path, _, _ := req.Option("path").String()
		depth, _, err := req.Option("depth").Int()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		res.SetOutput(obj.Tree(path, depth))
	},
	Type: []string{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			paths, ok := res.Output().([]string)
			if !ok {
				return nil, fmt.Errorf("wrong command response output type")
			}
			buf := new(bytes.Buffer)
			for _, s := range paths {
				fmt.Fprintln(buf, s)
			}
			return buf, nil
		},
	},
}

type treeList struct {
	Paths []string
}

func convertJsonToType(r io.Reader, format string) (node.Node, error) {
	switch format {
	case "cbor", "dag-cbor":
		return ipldcbor.FromJson(r)
	case "dag-pb", "protobuf":
		return nil, fmt.Errorf("protobuf handling in 'dag' command not yet implemented")
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}

func convertHexToType(r io.Reader, format string) ([]node.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	if data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	decd, err := hex.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	switch format {
	case "zec", "zcash":
		return zec.DecodeBlockMessage(decd)
	case "btc", "bitcoin":
		return btc.DecodeBlockMessage(decd)
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}

func convertRawToType(r io.Reader, format string) ([]node.Node, error) {
	switch format {
	case "eth":
		blk, txs, _, err := eth.FromRlpBlockMessage(r)
		if err != nil {
			return nil, err
		}

		var out []node.Node
		out = append(out, blk)
		for _, tx := range txs {
			out = append(out, tx)
		}
		/*
			for _, unc := range uncles {
				out = append(out, unc)
			}
		*/
		return out, nil
	default:
		return nil, fmt.Errorf("unknown target format: %s", format)
	}
}
