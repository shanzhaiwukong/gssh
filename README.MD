# GSSH

## Use
> go get github.com/shanzhaiwukong/gssh

## Demo
``` go
sh, err := gssh.New(ip, user, pwd, port)
if err != nil {
    painc(err)
}
err = sh.NewSession(60, 100)
if err != nil {
    painc(err)
}
defer sh.Close()
sh.BindInOutExitChannel(ch_in, ch_out, ch_exit)
// ...
```