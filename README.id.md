# Awd DriveRouter Desktop 💻☁️🔀

<p align="center">
  <img src="logo.png" width="130" height="130" alt="Awd DriveRouter Desktop Logo">
</p>

<p align="center">
  <a href="https://github.com/putuwahyu29/awd-drive-router-desktop/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="Badge Lisensi">
  </a>
  <img src="https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-blue?style=flat-square" alt="Badge Platform">
  <img src="https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Badge Go">
  <img src="https://img.shields.io/badge/Node.js-18%2B-339933?style=flat-square&logo=node.js&logoColor=white" alt="Badge Node.js">
  <img src="https://img.shields.io/badge/Framework-Wails%20v2-red?style=flat-square&logo=wails&logoColor=white" alt="Badge Wails">
  <img src="https://img.shields.io/badge/Storage-SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="Badge SQLite">
</p>

**Awd DriveRouter Desktop** adalah aplikasi desktop lintas platform yang kuat untuk menyatukan berbagai penyedia penyimpanan cloud ke dalam satu antarmuka manajemen file yang cerdas. Dibuat menggunakan **Wails (Go)** dan **React/Vite (TypeScript)**, aplikasi ini berfungsi sebagai lapisan *routing* pintar — memungkinkan Anda mengunggah, menyinkronkan, menjelajah, dan membagikan file di 17 opsi penyimpanan (Google Drive, OneDrive, Dropbox, Box, Yandex Disk, pCloud, MEGA, Koofr, MediaFire, 4Shared, Backblaze B2, SMB/LAN Share, FTP, SFTP, WebDAV, S3-compatible, dan Telegram) — semuanya dari satu aplikasi desktop.

---

## 🌐 Bahasa / Language
*   [Versi Bahasa Indonesia (Utama)](README.id.md)
*   [English Version](README.md)

---

## 📌 Daftar Isi
- [✨ Fitur Utama](#-fitur-utama)
- [📷 Tangkapan Layar](#-tangkapan-layar)
- [☁️ Penyedia Cloud yang Didukung](#️-penyedia-cloud-yang-didukung)
- [📁 Penyimpanan & Jalur Data](#-penyimpanan--jalur-data)
- [🚀 Panduan Penggunaan](#-panduan-penggunaan)
  - [Prasyarat](#prasyarat)
  - [Menghubungkan Akun Cloud](#menghubungkan-akun-cloud)
  - [Strategi Pengunggahan](#strategi-pengunggahan)
  - [Sinkronisasi Folder & Backup](#sinkronisasi-folder--backup)
  - [Membagikan Berkas](#membagikan-berkas)
  - [Arsip & Kompresi ZIP](#arsip--kompresi-zip)
- [🛠️ Panduan Pengembang](#️-panduan-pengembang)
  - [Struktur Direktori Proyek](#struktur-direktori-proyek)
  - [Menjalankan dalam Mode Pengembang](#menjalankan-dalam-mode-pengembang)
  - [Kompilasi Versi Produksi](#kompilasi-versi-produksi)
- [⚙️ Pemecahan Masalah & Log](#️-pemecahan-masalah--log)
- [⚠️ Penolakan Tanggung Jawab](#️-penolakan-tanggung-jawab)
- [📄 Lisensi](#-lisensi)

---

## ✨ Fitur Utama

*   **🔀 Multi-Cloud Router**: Hubungkan dan kelola banyak akun cloud storage secara bersamaan. DriveRouter secara cerdas mengarahkan pengunggahan file ke seluruh penyedia yang terkonfigurasi.
*   **📤 Strategi Unggah Pintar**: Pilih bagaimana file Anda didistribusikan di seluruh penyedia:
    *   **Round Robin**: Membagi unggahan secara merata bergiliran ke seluruh akun aktif.
    *   **Kapasitas Gratis Terbesar**: Selalu mengunggah ke akun yang memiliki sisa ruang penyimpanan paling besar.
*   **🔄 Sinkronisasi Folder & Otomatisasi Backup**: Buat tugas sinkronisasi yang menghubungkan folder lokal ke akun cloud mana pun.
    *   **Upload-Only Sync**: Menyalin folder lokal ke cloud secara aman tanpa mengubah file remote yang sudah ada.
    *   **Two-Way Sync**: Menjaga folder lokal dan remote selalu identik dua arah.
    *   **Interval Backup Fleksibel**: Pembackupan latar belakang berjalan otomatis sesuai interval yang ditentukan pengguna.
*   **📦 Kompresi & Arsip ZIP**: Pilih beberapa file dan kompres menjadi arsip `.zip` secara langsung. Hasil zip akan otomatis diunggah ke cloud tujuan.
*   **🔗 Penautan & Berbagi Berkas**: Ambil dan buka URL web asli dari berkas untuk dibagikan.
*   **🔒 Penyimpanan Kredensial Terenkripsi**: Token OAuth dan kredensial sensitif disimpan terenkripsi menggunakan lapisan keamanan bawaan.
*   **📊 Analisis Alokasi Penyimpanan**: Visualisasikan penggunaan ruang dan kuota seluruh akun dalam satu dasbor terpadu.
*   **⭐ Berkas Berbintang (Starred)**: Tandai berkas penting untuk akses cepat.
*   **🕒 Berkas Terkini (Recent)**: Tampilkan 30 berkas yang baru saja diubah.
*   **📡 Progres Unggah Real-time**: WebSocket bawaan mengirimkan progres pengunggahan secara langsung ke antarmuka aplikasi.
*   **⚙️ Integrasi Windows Native**: Dukungan system tray, minimize-to-tray, dan fitur *auto startup* saat Windows menyala.
*   **🌐 Antarmuka Dwibahasa**: Mendukung penuh Bahasa Indonesia dan Bahasa Inggris.

---

## 📷 Tangkapan Layar

| | |
|:---:|:---:|
| <img src="screenshot/home.png" width="400" alt="Dasbor Utama / Manajer File"/><br/>**Dasbor Utama / Manajer File** | <img src="screenshot/storage.png" width="400" alt="Analisis Penggunaan Penyimpanan"/><br/>**Analisis Penggunaan Penyimpanan** |
| <img src="screenshot/starred.png" width="400" alt="Berkas Berbintang"/><br/>**Berkas Berbintang** | <img src="screenshot/allocation.png" width="400" alt="Tampilan Alokasi Penyimpanan"/><br/>**Tampilan Alokasi Penyimpanan** |
| <img src="screenshot/config.png" width="400" alt="Pengaturan & Konfigurasi Penyedia"/><br/>**Pengaturan & Konfigurasi Penyedia** | <img src="screenshot/recent.png" width="400" alt="Berkas Terkini"/><br/>**Berkas Terkini** |

---

## ☁️ Penyedia Cloud yang Didukung

| Penyedia | Tipe Koneksi | Catatan |
| :--- | :---: | :--- |
| **Google Drive** | OAuth 2.0 | Membutuhkan OAuth client ID & secret dari Google Cloud Console |
| **OneDrive** | OAuth 2.0 | Menggunakan pendaftaran aplikasi Microsoft Entra (Azure AD) |
| **Dropbox** | OAuth 2.0 | Membutuhkan app key & secret dari Dropbox Developer |
| **Box** | OAuth 2.0 | Membutuhkan kredensial aplikasi Box OAuth 2.0 |
| **Yandex Disk** | OAuth 2.0 | Membutuhkan kredensial aplikasi Yandex OAuth |
| **pCloud** | OAuth 2.0 | Membutuhkan kredensial aplikasi pCloud OAuth |
| **MEGA** | Direct Login | Email & password — Dukungan E2EE terenkripsi |
| **Koofr** | Direct Login | Email & App Password — Kuota gratis 10 GB |
| **MediaFire** | Direct Login | Email & password — Kuota gratis 10 GB |
| **4Shared** | Direct Login | Email & password — Kuota gratis 15 GB |
| **Backblaze B2** | Access Keys | Application Key ID & Key — Kuota gratis 10 GB |
| **Windows Share (SMB)** | Direct Login | Host/IP, Nama Share, Username, dan Password — Penyimpanan LAN |
| **FTP Server** | Direct Login | Host, Port (21), username, dan password |
| **SFTP (SSH)** | Direct Login | Host, Port (22), username, dan password |
| **WebDAV** | Direct Login | URL Server, username, dan password / token aplikasi |
| **S3-Compatible** | Access Keys | Endpoint, bucket, access key ID, dan secret key |
| **Telegram Bot** | Bot Token | Bot token + ID Chat/Channel tujuan |
| **Telegram User** | MTProto | Nomor telepon, API ID, API hash — via `my.telegram.org` |

---

## 📁 Penyimpanan & Jalur Data

Semua konfigurasi, metadata, dan data sesi disimpan di direktori konfigurasi pengguna (`os.UserConfigDir()` — `%APPDATA%` di Windows):

*   **Database Aplikasi**: `<UserConfigDir>/driverouter/driverouter.db` — Database SQLite yang menyimpan rekaman file, kredensial terenkripsi, tugas sinkronisasi, dan pengaturan.
*   **Server OAuth Callback**: Berjalan lokal di `http://localhost:5998/oauth/callback` saat alur autentikasi.

---

## 🚀 Panduan Penggunaan

### Prasyarat
Untuk menjalankan aplikasi yang sudah dikompilasi, cukup klik dua kali berkas `Awd-DriveRouter.exe`. Tidak diperlukan driver tambahan selain Windows WebView2 (bawaan Windows 10/11).

### Menghubungkan Akun Cloud
1. Buka aplikasi dan pilih menu **Akun Cloud** (atau **Pengaturan** untuk penyedia OAuth).
2. Untuk penyedia OAuth (Google Drive, OneDrive, Dropbox, Box, Yandex Disk, pCloud):
   - Atur kredensial OAuth di menu **Pengaturan** terlebih dahulu.
   - Buka **Akun Cloud** dan klik **Hubungkan**.
   - Jendela peramban akan terbuka untuk autentikasi login.
3. Untuk penyedia direct login (MEGA, Koofr, MediaFire, FTP, SFTP, WebDAV, S3, Telegram):
   - Buka **Akun Cloud** → **Hubungkan** → pilih penyedia.
   - Isi kredensial yang diminta lalu kirim formulir.

---

---

## 🌐 Mode Penggunaan Aplikasi

Awd DriveRouter mendukung **Dual-Mode** fleksibel untuk memenuhi kebutuhan pengguna awam hingga advance:

### 1️⃣ Mode Desktop GUI (Pengguna Awam)
- **Cara Jalankan**: Cukup double-click `Awd-DriveRouter.exe` (atau jalankan `wails dev`).
- **Tampilan**: Antarmuka window desktop native biasa.

### 2️⃣ Mode Headless Web Server (Kompatibel Browser Web)
- **Cara Jalankan di Windows**:
  ```powershell
  .\driverouter.exe --server --port=8080 --api-key=rahasia123
  ```
- **Akses Peramban Web Lokal**: Buka peramban (Chrome / Edge / Firefox) di `http://localhost:8080`.
- **Akses dari Perangkat Lain di Jaringan Wi-Fi/LAN Internal**:
  - Aplikasi secara otomatis mencetak alamat IP LAN lokal Anda saat server di-start (contoh: `http://192.168.1.50:8080`).
  - Untuk melihat IP secara manual di Windows: buka PowerShell dan ketik `ipconfig` (lihat bagian *IPv4 Address*).
  - Perangkat lain (HP/Laptop) di jaringan Wi-Fi lokal dapat langsung mengakses `http://<IP-LAN-ANDA>:8080`.
- **Membatasi Hanya Untuk Komputer Ini Saja**:
  Jalankan dengan `--host=127.0.0.1` agar server tidak bisa diakses dari Wi-Fi lokal:
  ```powershell
  .\driverouter.exe --server --host=127.0.0.1 --port=8080
  ```
- **Integrasi System Tray**: Saat berjalan dalam mode headless di Windows, mengeklik *"Buka Awd DriveRouter"* pada icon System Tray akan **otomatis membuka peramban web di `http://localhost:8080`**.

---

## 🖥️ Panduan Deployment di VPS / Linux Server

Jika Anda menyewa VPS Linux (Ubuntu / Debian / CentOS / Cloud VM):

### Langkah 1: Kompilasi Binary Linux
Di PC lokal (Windows), buat binary Linux dengan fitur *Cross-Compilation*:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o driverouter-server .
```

### Langkah 2: Upload & Jalankan di VPS
Upload berkas `driverouter-server` dan folder `frontend/dist` (atau jalankan langsung dari source) ke VPS Anda.

Berikan izin eksekusi:
```bash
chmod +x driverouter-server
```

### Langkah 3: Menjalankan Latar Belakang (Systemd Service)
Agar aplikasi tetap aktif di VPS meskipun koneksi SSH ditutup, buat service Systemd di `/etc/systemd/system/driverouter.service`:

```ini
[Unit]
Description=Awd DriveRouter Headless Web Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/driverouter
ExecStart=/opt/driverouter/driverouter-server --server --port=8080 --api-key=rahasia123
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Aktifkan service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable driverouter
sudo systemctl start driverouter
```

---

## 🐳 Panduan Deployment Docker & Docker Compose

Untuk pengoperasian paling praktis tanpa perlu mengatur environment manual:

### Deployment 1-Klik:
```bash
docker-compose up -d --build
```
Akses antarmuka web melalui peramban di `http://ip-vps-anda:8080`.

---

## 🔒 Panduan Keamanan Deployment Publik

> [!CAUTION]
> Jika Anda hendak mempublikasikan Awd DriveRouter ke jaringan publik internet (VPS/Cloud), ikuti langkah pengamanan wajib berikut:

1. **Wajib Mengaktifkan API Key**:
   Jangan pernah menjalankan server di publik tanpa mengisi `API_KEY` atau flag `--api-key`. Tanpa API Key, siapa pun yang mengetahui IP server Anda dapat mengontrol akun cloud Anda.
   ```yaml
   environment:
     - SERVER_MODE=true
     - API_KEY=KataSandiAcakYangKuat123!
   ```

2. **Wajib Menggunakan HTTPS / TLS (Reverse Proxy Nginx / Caddy)**:
   Jangan membuka port HTTP mentah (`:8080`) langsung ke publik. Gunakan Reverse Proxy seperti **Nginx** atau **Caddy** dengan sertifikat SSL gratis dari Let's Encrypt.

   **Contoh Konfigurasi Nginx (`/etc/nginx/sites-available/driverouter`)**:
   ```nginx
   server {
       listen 80;
       server_name drive.domainanda.com;
       return 311 https://$host$request_uri;
   }

   server {
       listen 443 ssl http2;
       server_name drive.domainanda.com;

       ssl_certificate /etc/letsencrypt/live/drive.domainanda.com/fullchain.pem;
       ssl_certificate_key /etc/letsencrypt/live/drive.domainanda.com/privkey.pem;

       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_http_version 1.1;
           proxy_set_header Upgrade $http_upgrade;
           proxy_set_header Connection "upgrade";
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

3. **Atur Firewall Server**:
   Tutup port `8080` dari akses publik luar menggunakan UFW/iptables, sehingga aplikasi hanya diakses secara internal oleh Nginx melalui HTTPS (Port `443`):
   ```bash
   sudo ufw default deny incoming
   sudo ufw allow 80/tcp
   sudo ufw allow 443/tcp
   sudo ufw allow 22/tcp
   sudo ufw enable
   ```

---

## 🛠️ Panduan Pengembang

### Kompilasi Mode Pengembang
```bash
wails dev
```

### Kompilasi Versi Produksi Windows
```bash
wails build
```

---

## 📄 Lisensi
Hak Cipta © 2026 Putu Wahyu. Dilisensikan di bawah [Lisensi MIT](LICENSE).
