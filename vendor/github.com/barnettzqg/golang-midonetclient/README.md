# golang-midonetclient
midonet client write in golang

# 版本说明
支持midonet api v1，v5

```
  c, err := midonetclient.NewClient(&types.MidoNetAPIConf{})
	if err != nil {
		conf.Log.Error("Create midonet client error.", err.Error())
		return nil, err
	}
  c.CreateTenant(&types.Tenant{})
```
