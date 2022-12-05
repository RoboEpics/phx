# Phoenix CLI
Run your data science workloads on high-performance cloud infrastructure in the fewest of steps.

# Table of contents
- [Installation](#installation)
- [Usage](#usage)
- [Contact](#contact)

# Installation

You can download and install the CLI for any of the following platforms.

| Supported Platform | Download (Latest)            |
|--------------------|------------------------------|
| Windows            | [Link][latest-windows-amd64] |
| Linux              | [Link][latest-linux-amd64]   |
| macOS              | [Link][latest-macos-amd64]   |

[latest-windows-amd64]: https://github.com/RoboEpics/phoenix-binaries/releases/download/v0.4.0/phx-windows-amd64.exe
[latest-linux-amd64]: https://github.com/RoboEpics/phoenix-binaries/releases/download/v0.4.0/phx-linux-amd64
[latest-macos-amd64]: https://github.com/RoboEpics/phoenix-binaries/releases/download/v0.4.0/phx-darwin-amd64

To download a specific version, visit the [releases page](https://github.com/RoboEpics/phoenix-binaries/releases).

# Usage

First you need to initialize Phoenix in your data science project directory:

```bash
phx init
```

This will create a `.phoenix` folder inside your project root directory.

Next you need to login to the Phoenix Platform by running this command and filling out your credentials in the prompts:

```bash
phx login
```

If you need to login with a static token, you can pass the `--static` flag to `phx login`:
```bash
phx login --static
```

As easily as that, now your project is ready for the cloud.

## Running jobs

You can run a job:

```bash
phx run --cluster $CLUSTER_NAME --flavor $FLAVOR_NAME --name $YOUR_JOB_NAME $COMMAND $ARGS
```

## Creating Jupyter Notebooks

You can also run a Jupyter Notebook on-demand and attach it to Google Colab as an external powerful non-interrupting runtime kernel:

```bash
phx jupyter create --cluster $CLUSTER_NAME --flavor $FLAVOR_NAME --name $YOUR_JUPYTER_INSTANCE_NAME
phx jupyter attach
```

Now, as long as your terminal is open, you can connect your Colab to this runtime using the "Connect to a local runtime" button in Colab interface.

You can read more about how to connect Colab to a local runtime [here](https://research.google.com/colaboratory/local-runtimes.html).

# Contact
If you had any questions or problems, join our server on [**Discord**](https://discord.gg/8DMfjmn6gc).

