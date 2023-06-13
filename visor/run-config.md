





## *RunConfig*
Root of the config file


### Fields

<dl>
<dt>
	<code>name</code>  <strong>string</strong>  - required
</dt>

<dd>

Name of the upgrade.


<blockquote>It is recommended to use the Vega version you wish to upgrade to as the name. These can be found in the releases list of the Vega Github repository.</blockquote>
</dd>

<dt>
	<code>vega</code>  <strong><a href="#vegaconfig">VegaConfig</a></strong>  - required
</dt>

<dd>

Configuration of a Vega node.

</dd>

<dt>
	<code>data_node</code>  <strong><a href="#datanodeconfig">DataNodeConfig</a></strong>  - optional
</dt>

<dd>

Configuration of a data node.

</dd>



### Complete example


```hcl
name = "v1.65.0"

[vega]
 [vega.binary]
  path = "/path/vega-binary"
  args = ["--arg1", "val1", "--arg2"]
 [vega.rpc]
  socketPath = "/path/socket.sock"
  httpPath = "/rpc"

```


</dl>

---


## *VegaConfig*
Configuration options for the Vega binary and its arguments.


### Fields

<dl>
<dt>
	<code>binary</code>  <strong><a href="#binaryconfig">BinaryConfig</a></strong>  - required
</dt>

<dd>

Configuration of Vega binary and arguments required to run it.

</dd>

<dt>
	<code>rpc</code>  <strong><a href="#rpcconfig">RPCConfig</a></strong>  - required
</dt>

<dd>

Visor communicates with the core node via RPC API that runs over a UNIX socket.
This parameter configures the UNIX socket to match the core node configuration. This value can be found in the config.toml file used by the core node under the heading [Admin.Server]


</dd>



### Complete example


```hcl
[vega]
 [vega.binary]
  path = "/path/vega-binary"
  args = ["--arg1", "val1", "--arg2"]
 [vega.rpc]
  socketPath = "/path/socket.sock"
  httpPath = "/rpc"

```


</dl>

---


## *DataNodeConfig*
Configures the data node binary and its arguments.


### Fields

<dl>
<dt>
	<code>binary</code>  <strong><a href="#binaryconfig">BinaryConfig</a></strong>  - required
</dt>

<dd>



</dd>



### Complete example


```hcl
[data_node]
 [data_node.binary]
  path = "/path/data-node-binary"
  args = ["--arg1", "val1", "--arg2"]

```


</dl>

---


## *BinaryConfig*
Configures the data node binary and its arguments.


### Fields

<dl>
<dt>
	<code>path</code>  <strong>string</strong>  - required
</dt>

<dd>

Path to the data node binary.


<blockquote>Both absolute or relative path can be used.
Relative path is relative to a parent folder of this config file.
</blockquote>
</dd>

<dt>
	<code>args</code>  <strong>[]string</strong>  - required
</dt>

<dd>

Arguments that will be applied to the binary.


<blockquote>Each element the list represents one space separated argument. An argument and its value should be in separate elements.
</blockquote>
</dd>



### Complete example


```hcl
path = "/path/binary"
args = ["--arg1", "val1", "--arg2"]

```


</dl>

---


## *RPCConfig*
Configures the connection to the core node exposed UNIX socket RPC API. These values can be found in the `config.toml` file used by the core node under the heading `[Admin.Server]`


### Fields

<dl>
<dt>
	<code>socketPath</code>  <strong>string</strong>  - required
</dt>

<dd>
Path of the mounted socket.
</dd>

<dt>
	<code>httpPath</code>  <strong>string</strong>  - required
</dt>

<dd>
HTTP path of the socket path.
</dd>



### Complete example


```hcl
[vega.rpc]
 socketPath = "/path/socket.sock"
 httpPath = "/rpc"

```


</dl>

---


