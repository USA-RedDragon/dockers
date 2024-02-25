# SteamCMD

## What is SteamCMD?

The Steam Console Client or SteamCMD is a command-line version of the Steam client. Its primary use is to install and
update various dedicated servers available on Steam using a command-line interface. It works with games that use the
SteamPipe content system. All games have been migrated from the deprecated HLDSUpdateTool to SteamCMD. This image can be
used as a base image for Steam-based dedicated servers (Source: [developer.valvesoftware.com](https://developer.valvesoftware.com/wiki/SteamCMD)).

## How to use this image

Whilst it's recommended to use this image as a base image of other game servers, you can also run it in an interactive
shell using the following command:

```bash
docker run -it --name=steamcmd ghcr.io/usa-reddragon/steamcmd bash
./steamcmd.sh +force_install_dir /home/steam/squad-dedicated +login anonymous +app_update 403240 +quit
```

This can prove useful if you are just looking to test a certain game server installation.

Running with named volumes:

```bash
docker volume create steamcmd_login_volume # Optional: Location of login session
docker volume create steamcmd_volume # Optional: Location of SteamCMD installation

docker run -it \
    -v "steamcmd_login_volume:/home/steam/Steam" \
    -v "steamcmd_volume:/home/steam/steamcmd" \
    ghcr.io/usa-reddragon/steamcmd bash
```

This setup is necessary if you have to download a non-anonymous appID or upload a steampipe build. For an example check out:
<https://hub.docker.com/r/cm2network/steampipe/>

## Configuration

This image includes the `nano` text editor for convenience.

The `steamcmd.sh` can be found in the following directory: `/home/steam/steamcmd`
