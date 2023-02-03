





## *VisorConfigFile*
Root of the config file


### Fields

<dl>
<dt>
	<code>maxNumberOfFirstConnectionRetries</code>  <strong>int</strong>  - optional
</dt>

<dd>

Visor communicates with the core node via RPC API.
This variable allows a validator to specify how many times Visor should try to establish a connection to the core node before the Visor process fails.
The `maxNumberOfFirstConnectionRetries` is only taken into account during the first start up of the Core node process - not restarts.



Default value: <code>10</code>

<blockquote>There is a 2 second delay between each attempt. Setting the max retry number to 5 means Visor will try to establish a connection 5 times in 10 seconds.
</blockquote>
</dd>

<dt>
	<code>maxNumberOfRestarts</code>  <strong>int</strong>  - optional
</dt>

<dd>

Visor starts and manages the processes of provided binaries.
This allows a user to define the maximum number of restarts in case any of
the processes have failed before the Visor process fails.



Default value: <code>3</code>

<blockquote>The amount of time Visor waits between restarts can be set by `restartsDelaySeconds`.
</blockquote>
</dd>

<dt>
	<code>restartsDelaySeconds</code>  <strong>int</strong>  - optional
</dt>

<dd>

Number of seconds that Visor waits before it tries to re-start the processes.



Default value: <code>5</code>
</dd>

<dt>
	<code>stopSignalTimeoutSeconds</code>  <strong>int</strong>  - optional
</dt>

<dd>

Number of seconds that Visor waits after it sends termination signal (SIGTERM) to running processes.
After the time has elapsed Visor force-kills (SIGKILL) any running processes.



Default value: <code>15</code>
</dd>

<dt>
	<code>upgradeFolders</code>  <strong>map[string]string</strong>  - optional
</dt>

<dd>

During the upgrade, by default Visor looks for a folder with a name identical to the upgrade version.
The default behaviour can be changed by providing mapping between `version` and `custom_folder_name`.
If a custom mapping is provided, during the upgrade Visor uses the folder given in the mapping for specific version.


</dd>

<dt>
	<code>autoInstall</code>  <strong><a href="#autoinstallconfig">AutoInstallConfig</a></strong>  - required
</dt>

<dd>

Allows to define assets that should be automatically downloaded from Github for a specific release.


</dd>



### Complete example


```hcl
maxNumberOfRestarts = 3
restartsDelaySeconds = 5

[upgradeFolders]
 "vX.X.X" = "vX.X.X"

[autoInstall]
 enabled = false

```


</dl>

---


## *AutoInstallConfig*
Allows to define assets that should be automatically downloaded from Github for a specific release.


### Fields

<dl>
<dt>
	<code>enabled</code>  <strong>bool</strong>  - required
</dt>

<dd>

Whether or not the autoinstall should be used


Default value: <code>true</code>
</dd>

<dt>
	<code>repositoryOwner</code>  <strong>string</strong>  - required
</dt>

<dd>

Owner of the repository from where the assets should be downloaded.


Default value: <code>vegaprotocol</code>
</dd>

<dt>
	<code>repository</code>  <strong>string</strong>  - required
</dt>

<dd>

Name of the repository from where the assets should be downloaded.


Default value: <code>vega</code>
</dd>

<dt>
	<code>assets</code>  <strong><a href="#assetsconfig">AssetsConfig</a></strong>  - required
</dt>

<dd>

Definitions of the assets that should be downloaded from the Github repository.

</dd>



### Complete example


```hcl
[autoInstall]
 enabled = true
 repositoryOwner = "vegaprotocol"
 repository = "vega"
 [autoInstall.assets]
  [autoInstall.assets.vega]
   assset_name = "vega-darwin-amd64.zip"
   binary_name = "vega"

```


</dl>

---


## *AssetsConfig*


### Fields

<dl>
<dt>
	<code>vega</code>  <strong><a href="#asset">Asset</a></strong>  - required
</dt>

<dd>

Allows to define asset name to download Vega binary.

</dd>

<dt>
	<code>data_node</code>  <strong><a href="#asset">Asset</a></strong>  - optional
</dt>

<dd>

Allows to define asset name to download data node binary.

</dd>



</dl>

---


## *Asset*


### Fields

<dl>
<dt>
	<code>assset_name</code>  <strong>string</strong>  - required
</dt>

<dd>

Name of the asset on Github.

</dd>

<dt>
	<code>binary_name</code>  <strong>string</strong>  - optional
</dt>

<dd>

Binary name definition can be used in case the asset is a zip file and the binary is included inside of it.


</dd>



</dl>

---


