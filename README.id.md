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
