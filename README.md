# Liburday

**Liburday** provides a simple and accessible list of Indonesia's public holidays and observances in JSON format.

## Usage

To retrieve the list of holidays for a specific year, use the following `curl` command:
```curl
curl https://raw.githubusercontent.com/chay22/liburday/refs/heads/main/2025.json
```

Each JSON file (e.g. `2025.json`) contains an array of objects with the following structure:

```json
[
  {
    "name": "Tahun Baru Masehi",
    "date": "2025-01-01",
    "is_national": 1
  }
]
```
- `name`: Name of the holiday or observance.
- `date`: Date in YYYY-MM-DD format.
- `is_national`:
  - `1` for official national holidays.
  - `0` for non-national observances such as Diwali or other cultural/religious events.

## Data Sources
Holiday and observance data are aggregated from:

- [Tanggalan.com](https://tanggalan.com/)
- Google Calendar API
