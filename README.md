# getemoji 🎨😀

**getemoji** is a command-line interface (CLI) tool that allows you to generate emoji images in SVG or PNG format.
It downloads the emoji from twemoji and can optionally resize it to a specified size.

## ✨ Features

- Generate emoji images in SVG or PNG format
- Resize emojis to a specific size (for PNG output)
- Support for emoji shortcodes and Unicode characters
- Configurable via command-line flags, environment variables, or a YAML config file

## 🚀 Installation

There are two ways to install **getemoji**:

1. Using Go:
   To install **getemoji**, you need to have Go installed on your system. Then, you can use the following command:

   ```
   go install github.com/igolaizola/getemoji@latest
   ```

2. Downloading pre-built binaries:
   You can download pre-built binaries for your operating system from the [Releases](https://github.com/igolaizola/getemoji/releases) page of the GitHub repository.

## 🛠 Usage

The basic syntax for using **getemoji** is:

```
getemoji --emoji <emoji> --size <size> --output <output>
```

### 🚩 Flags

- `-emoji`: The emoji to generate (required)
- `-size`: The size of the output image in pixels (required for PNG output)
- `-output`: The output file name (optional, defaults to `icon.svg` or `icon<size>.png`)
- `-config`: Path to a YAML configuration file (optional)

### 🌿 Environment Variables

You can also use environment variables to set the flags. Prefix the flag name with `GETEMOJI_`. For example:

```
GETEMOJI_EMOJI="smile" GETEMOJI_SIZE=64 GETEMOJI_OUTPUT="smile.png" getemoji
```

### 📚 Examples

1. Generate an SVG of a smiley face emoji:

   ```
   getemoji -emoji "smile" -output smile.svg
   ```

2. Generate a 64x64 PNG of a heart emoji:

   ```
   getemoji -emoji "❤️" -size 64 -output heart.png
   ```

3. Use a configuration file:

   ```
   getemoji -config config.yaml
   ```

   Example `config.yaml`:

   ```yaml
   emoji: "rocket"
   size: 128
   output: rocket.png
   ```

## 🔢 Version Information

You can check the version of **getemoji** by running:

```
getemoji version
```

This will display the version number, commit hash, and build date (if available).

## 🔗 Dependencies

**getemoji** uses the following open-source libraries:

- [github.com/kyokomi/emoji/v2](https://github.com/kyokomi/emoji)
- [github.com/srwiley/oksvg](https://github.com/srwiley/oksvg)
- [github.com/srwiley/rasterx](https://github.com/srwiley/rasterx)
- [github.com/peterbourgon/ff/v3](https://github.com/peterbourgon/ff)

## 📜 License

**getemoji** is licensed under the MIT License.

The emoji graphics are from [twemoji](https://github.com/twitter/twemoji) and are downloaded using `https://cdn.jsdelivr.net/gh/jdecked/twemoji`.
These graphics are licensed under the Creative Commons Attribution 4.0 International License (CC-BY-4.0).
To view a copy of this license, visit [http://creativecommons.org/licenses/by/4.0/](http://creativecommons.org/licenses/by/4.0/).

## 🤝 Contributing

Contributions to **getemoji** are welcome! Please feel free to submit a Pull Request.

## 💬 Support

If you encounter any problems or have any questions about **getemoji**, please open an issue on the GitHub repository.
