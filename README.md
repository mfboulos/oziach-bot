# OziachBot ![language](https://img.shields.io/badge/Go-1.13-lightblue.svg?cacheSeconds=2592000) [![Actions Status](https://github.com/mfboulos/oziachbot/workflows/build/badge.svg)](https://github.com/mfboulos/oziachbot/actions)
OziachBot is the first ever open source Twitch IRC bot for Old School Runescape.

## Usage
OziachBot will soon be available to connect to your channel via a simple OAuth authentication. It will need minimal permissions, and simply serves as affirmation that you're the owner of the channel you want OziachBot to connect to. From there, you will be able to connect and disconnect OziachBot from your channel at will.

## Features
All OziachBot features are documented below.

### !level
Queries the Old School Runescape Hiscore API for the level and experience values of `skill`. Extracts `username` rank based on current game mode (normal, ironman, etc.)

Aliases: `!lvl`

Args: `[skill] [username]`

### !total
Alias for `!level Overall [username]`

Aliases: `!overall`

Args: `[username]`

## Contributing
If you'd like to contribute, take a look at the [contributing guidelines](CONTRIBUTING.md)

## Licensing
OziachBot is licensed under the GNU General Public License v3.