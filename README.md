# Phoenix CLI
Run your data science workloads on high-performance cloud infrastructure in the fewest of steps.

# Table of contents
- [Installation](#installation)
  - [Download](#download)
  - [Setup](#setup)
    - [Linux](#linux)
    - [Windows](#windows)
    - [macOS](#macos)
- [Usage](#usage)
- [Contact](#contact)

# Installation

## Download

You can download and install the CLI for any of the following platforms.

| Supported Platform | Download (Latest)            |
|--------------------|------------------------------|
| Windows            | [Link][latest-windows-amd64] |
| Linux              | [Link][latest-linux-amd64]   |
| macOS              | [Link][latest-macos-amd64]   |

[latest-windows-amd64]: https://github.com/RoboEpics/phx/releases/download/v0.4.0/phx-windows.exe
[latest-linux-amd64]: https://github.com/RoboEpics/phx/releases/download/v0.4.0/phx-linux
[latest-macos-amd64]: https://github.com/RoboEpics/phx/releases/download/v0.4.0/phx-darwin

To download a specific version, visit the [releases page](https://github.com/RoboEpics/phoenix-binaries/releases).

## Setup

### Linux

Run the following commands in the directory to make the CLI accessible from your terminal:

```bash
mkdir ~/bin
mv phx-linux ~/bin/phx
echo "export PATH=\"\$PATH:\$HOME/bin\"" >> ~/.bashrc
```

### Windows

To make the CLI accessible from your terminal, first create a folder called `bin` in your home directory,
then open the environment variables settings by searching `environment variables` in the Start Menu and
clicking on `Edit the system environment variables` and then clicking on `Environment Variables`.

In the opened window, find `Path` in the `User variables` list and double click on it. Then click on `New` to create a new entry in the list.

Paste the following line in the newly created entry:

```
%USERPROFILE%\bin
```

Lastly, click `OK` for all the opened windows.

### macOS

Run the following commands to make the CLI accessible from your terminal:

```zsh
mkdir ~/bin
mv phx-darwin ~/bin/phx
echo "export PATH=\"\$PATH:\$HOME/bin\"" >> ~/.zshrc
```

# Usage

First you need to initialize Phoenix in your project directory:

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

