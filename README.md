# CouchTube

CouchTube is a self-hostable YouTube frontend designed to simulate a TV channel experience. It dynamically loads YouTube videos from a predefined list of channels and schedules playback based on the current time. Users can also submit their custom video lists through a JSON file URL.

I hope CouchTube will be a community-driven project, where people can create and share their own channel JSON lists. Feel free to submit pull requests with new channel lists to enhance the default set in this repo.

The project is in its early days of development. There will probably be many issues and bugs. Please use [issues](https://github.com/ozencb/couchtube/issues) to report them.

CouchTube is inspired by [ytch.xyz](https://ytch.xyz/).


---

## Getting Started

### Using Docker

To run CouchTube using Docker, you can either use a `docker-compose.yml` file:

```yaml
version: "3.8"

services:
  couchtube:
    image: ghcr.io/ozencb/couchtube:latest
    container_name: couchtube_app
    ports:
      - "8363:8363"  
    environment:
      - PORT=8363
      - READONLY_MODE=false
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8363"]
      interval: 30s
      timeout: 10s
      retries: 3

```

Or start it directly with a `docker run` command:

```sh
docker run -d \
  --name couchtube_app \
  -p 8363:8363 \
  -e PORT=8363 \
  -e READONLY_MODE=false \
  --restart unless-stopped \
  --health-cmd="curl -f http://localhost:8363 || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  ghcr.io/ozencb/couchtube:latest

```

### Building From Source

Ensure you have Golang 1.22 or higher installed. Create a `.env` file with the same environment variables found in `docker-compose.yml`.

```dotenv
PORT=8363
DATABASE_FILE_PATH=/app/data/couchtube.db
READONLY_MODE=false
```

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/ozencb/couchtube.git
   cd couchtube
   ```

2. **Install Go Dependencies**:
   ```sh
   go mod tidy
   ```

3. **Run the Application**:
   ```sh
   go run main.go
   ```
   The server will start on `http://localhost:8363`.

4. **Access the Application**:
   Open a browser and go to `http://localhost:8363` to access CouchTube.

On the first run, CouchTube will create a `couchtube.db` SQLite database file, initialize necessary tables, and populate them with any default channels found in `videos.json`.


---

## Usage

CouchTube loops through a channel's videos and only shows the section of the video marked by `sectionStart` and `sectionEnd`. The scheduler aims to distribute these videos throughout the day, so two different users should see the same video for a given channel.

### Environment Variables

You can configure CouchTube using environment variables.

| Variable             | Description                                                                 |
| -------------------- | --------------------------------------------------------------------------- |
| `PORT`               | The port number on which CouchTube will run.                                |
| `DATABASE_FILE_PATH` | The path to the SQLite database file used by CouchTube.                     |
| `JSON_FILE_PATH`     | The path to the JSON file used by CouchTube.                                |
| `FULL_SCAN`          | Overwrites the existing data in the DB with the videos in JSON file.        |
| `READONLY_MODE`      | If set to `true`, CouchTube will run in read-only mode, preventing changes. |


### Custom JSON Format for Channel and Video Lists

You can create custom JSON files to specify channels and video lists.

#### JSON Structure

Create your JSON file using the following format:

```json
{
  "channels": [
    {
      "name": "Channel Name",
      "videos": [
        {
          "id": "VIDEO_ID",
          "sectionStart": 10,
          "sectionEnd": 300
        },
        {
          "id": "ANOTHER_VIDEO_ID",
          "sectionStart": 0,
          "sectionEnd": 200
        }
      ]
    },
    {
      "name": "Another Channel Name",
      "videos": [
        {
          "id": "DIFFERENT_VIDEO_ID",
          "sectionStart": 0,
          "sectionEnd": 150
        }
      ]
    }
  ]
}
```

#### Field Descriptions

- **channels**: An array of channel objects. Each channel contains:
  - **name**: The channel name.
  - **videos**: An array of video objects containing:
    - **id**: The ID of the YouTube video.
    - **sectionStart**: The start time (in seconds) within the video where playback begins.
    - **sectionEnd**: The end time (in seconds) within the video where playback ends.


Save your custom JSON file using the above structure or make it accessible through a URL.

### Uploading Custom JSON

Within the CouchTube application, click the settings icon (gear icon) to submit a URL pointing to your custom JSON file. This URL should contain the JSON with channels and videos you want CouchTube to use.

---

## Contributing

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/YourFeature`).
3. Commit your changes (`git commit -m 'Add some feature'`).
4. Push to the branch (`git push origin feature/YourFeature`).
5. Open a Pull Request.

---

## License

This project is licensed under the GNU General Public License.

---

## Additional Notes

1. **Database**: Ensure you don't have an existing `couchtube.db` file to avoid database conflicts.
2. **Error Handling**: The app includes basic error handling but might need enhancements for production use.
3. **Video Availability**: Videos marked private, restricted, or disabled for embedding may not play. CouchTube attempts to handle such errors by skipping to the next available video.

---

Enjoy using CouchTube!