package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	rpc "github.com/hekmon/transmissionrpc"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ClientKey = "rpcClient"
)

func getClient(vp *viper.Viper) *rpc.Client {
	return vp.Get(ClientKey).(*rpc.Client)
}

func printTorrent(t *rpc.Torrent) {
	fmt.Println(torrentToString(t))
}

func torrentToString(t *rpc.Torrent) string {
	var id string
	if t.ID != nil {
		id = strconv.FormatInt(*t.ID, 10)
	} else {
		id = "-"
	}
	var progress string
	if t.PercentDone != nil {
		value := int(100 * *t.PercentDone)
		if value == 100 {
			progress = "OK!"
		} else {
			progress = fmt.Sprintf("%2d%%", value)
		}
	} else {
		progress = "---"
	}

	var available string = "---"
	if t.HaveValid != nil && t.HaveUnchecked != nil && t.DesiredAvailable != nil && t.LeftUntilDone != nil {
		have := *t.HaveValid + *t.HaveUnchecked
		avail := have + *t.DesiredAvailable
		all := have + *t.LeftUntilDone
		if av := 100 * avail / all; av == 100 {
			available = "OK!"
		} else {
			available = fmt.Sprintf("%2d%%", av)
		}
	}

	var status string
	if t.Status != nil {
		switch *t.Status {
		case rpc.TorrentStatusStopped:
			status = "||"
		case rpc.TorrentStatusDownload, rpc.TorrentStatusDownloadWait,
			rpc.TorrentStatusCheck, rpc.TorrentStatusCheckWait:
			status = ">>"
		case rpc.TorrentStatusSeed, rpc.TorrentStatusSeedWait:
			status = "<<"
		}
	} else {
		status = "--"
	}

	var name string
	if t.Name != nil {
		name = *t.Name
	} else {
		name = "-"
	}
	return fmt.Sprintf("[ %4s ][ %s ][ %s ][ %s ] %s", id, available, progress, status, name)
}

func argsToIDs(args []string) ([]int64, error) {
	var ids []int64
	for _, arg := range args {
		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("fail to parse id=%s err=%w", arg, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func NewAddCommand(vp *viper.Viper) *cobra.Command {
	deleteTorrent := new(bool)
	detail := new(bool)
	cmd := &cobra.Command{
		Use:   "add [FILE or URLS]",
		Short: "Add torrent file or magnet link",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient(vp)
			for _, arg := range args {
				var torrent *rpc.Torrent
				var err error
				if strings.HasPrefix(arg, "http") {
					torrent, err = client.TorrentAdd(&rpc.TorrentAddPayload{
						Filename: &arg,
					})
				} else {
					torrent, err = client.TorrentAddFile(arg)
					if err == nil && *deleteTorrent {
						if err := os.Remove(arg); err != nil {
							return fmt.Errorf("fail to remove torrent file=%s err=%w", arg, err)
						}
					}
				}
				if err != nil {
					return fmt.Errorf("fail to add torrent arg=%q err=%w", arg, err)
				}
				if *detail {
					printTorrent(torrent)
				} else {
					fmt.Println(*torrent.ID)
				}
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(detail, "detail", false, "Show details of added torrent")
	flags.BoolVar(deleteTorrent, "delete", false, "Delete torrent file on successful addition")
	return cmd
}

func NewListCommand(vp *viper.Viper) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List current torrents",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient(vp)
			ids, err := argsToIDs(args)
			if err != nil {
				return err
			}
			torrents, err := client.TorrentGetAllFor(ids)
			if err != nil {
				return fmt.Errorf("fail to get torrents err=%w", err)
			}
			for _, torrent := range torrents {
				printTorrent(torrent)
			}
			return nil
		},
	}
	return cmd
}

func isDone(t *rpc.Torrent) bool {
	if t.Status != nil && *t.Status == rpc.TorrentStatusStopped {
		if t.LeftUntilDone != nil && *t.LeftUntilDone == 0 {
			return true
		}
	}
	return false
}

func NewRemoveCommand(vp *viper.Viper) *cobra.Command {
	var delete bool
	cmd := &cobra.Command{
		Use:   "remove [TORRENT IDs]",
		Short: "Remove specified torrents or already finished and stopped torrents",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getClient(vp)
			ids, err := argsToIDs(args)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				torrents, err := client.TorrentGetAllFor(nil)
				if err != nil {
					return fmt.Errorf("fail to get torrents err=%w", err)
				}
				for _, t := range torrents {
					if isDone(t) {
						ids = append(ids, *t.ID)
					}
				}
			}
			if len(ids) > 0 {
				if err := client.TorrentRemove(&rpc.TorrentRemovePayload{
					IDs:             ids,
					DeleteLocalData: delete,
				}); err != nil {
					return err
				}
			}
			for _, id := range ids {
				fmt.Println(id)
			}
			return nil
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&delete, "delete", false, "Delete downloaded files")
	return cmd
}

func main() {
	vp := viper.New()
	vp.SetEnvPrefix("TRANSMISSION")
	vp.AutomaticEnv()
	vp.SetConfigName("config")
	vp.SetConfigType("yaml")
	vp.AddConfigPath(os.ExpandEnv("$HOME/.tmc"))

	root := &cobra.Command{
		Use:          os.Args[0],
		Short:        "Transmission Client (CUI)",
		SilenceUsage: true,
	}
	flags := root.PersistentFlags()
	flags.StringP("host", "s", "localhost", "Transmission server host")
	flags.IntP("port", "p", 0, "Transmission server port")
	flags.String("url", "", "Transmission RPC URL (optional)")
	flags.StringP("user", "u", "", "Transmission user name")
	flags.StringP("password", "w", "", "Transmission password")
	flags.BoolP("https", "t", false, "Use TLS for connection")
	flags.String("useragent", "TorrentCLI", "UserAgent name for HTTP Client")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var config rpc.AdvancedConfig

		host := vp.GetString("host")
		user := vp.GetString("user")
		pass := vp.GetString("password")
		config.HTTPS = vp.GetBool("https")
		config.Port = vp.GetUint16("port")
		config.UserAgent = vp.GetString("useragent")
		rpcUrl := vp.GetString("url")
		if len(rpcUrl) > 0 {
			urlObj, err := url.Parse(rpcUrl)
			if err != nil {
				return err
			}
			host = urlObj.Host
			if info := urlObj.User; info != nil {
				user = info.Username()
				if p, ok := info.Password(); ok {
					pass = p
				}
			}
			config.HTTPS = urlObj.Scheme == "https"
			if s := urlObj.Port(); len(s) > 0 {
				v, err := strconv.ParseUint(s, 10, 16)
				if err != nil {
					return err
				}
				config.Port = uint16(v)
			}
			if uri := urlObj.RequestURI(); len(uri) > 0 {
				config.RPCURI = uri
			}
		}
		if config.HTTPS && config.Port == 0 {
			config.Port = 443
		}
		if len(user) > 0 && len(pass) == 0 {
			prompt := promptui.Prompt{
				Label: "Password",
				Mask:  '*',
			}
			if result, err := prompt.Run(); err != nil {
				return err
			} else {
				pass = result
			}
		}
		client, err := rpc.New(host, user, pass, &config)
		if err != nil {
			return err
		}
		vp.Set(ClientKey, client)
		return err
	}
	vp.MergeInConfig()
	vp.BindPFlags(flags)
	root.AddCommand(NewAddCommand(vp))
	root.AddCommand(NewListCommand(vp))
	root.AddCommand(NewRemoveCommand(vp))
	root.AddCommand(&cobra.Command{
		Use:   "save",
		Short: "Save current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			file := os.ExpandEnv("$HOME/.tmc/config.yml")
			dir := path.Dir(file)
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Save configuration to", file)
			return vp.WriteConfigAs(file)
		},
	})
	root.Execute()
}
