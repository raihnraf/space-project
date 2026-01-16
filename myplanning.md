here's the project that i wanna create, u can ask whatever you need to ask, so the project became very clear

The Project: "OrbitStream: High-Throughput Satellite Telemetry Engine"
The Concept: A system that simulates a constellation of satellites sending thousands of health-check signals (temperature, voltage, position) per second, ingests them into a time-series database, and visualizes anomalies.


OrbitStream (Satellite Telemetry Pipeline) | Go, Python, TimescaleDB, Docker

Designed a high-throughput ground station simulator capable of ingesting 10,000+ telemetry points/second.

Implemented TimescaleDB hypertables for time-series partitioning, reducing query latency by 40% for historical datasets.

Built a Go-based ingestion service with batch processing to handle burst data loads from simulated satellite constellations.

Automated anomaly detection (voltage/temperature spikes) visualized on Grafana dashboards.


saya prompt ai dan dijawab detailnya di bawah ini:

sistem ini seperti Pipa Air.

Airnya = Data Telemetri (Suhu, Baterai, Posisi Satelit).

Keran Air = Python Script (Dia yang nyalain air/ngirim data terus-terusan).

Pipa Penampung = Golang App (Dia yang nerima air deras dan masukin ke tangki).

Tangki Air = Database (TimescaleDB).

Meteran Air = Dashboard (Grafana).

Berikut alur kerjanya step-by-step dan cara nge-test-nya:

1. Arsitektur: Siapa Melakukan Apa?
Kamu tidak akan klik-klik tombol di browser (seperti di web Laravel). Kamu akan menjalankan program di Terminal.

Script Python (The Spammer):

Tugas: Pura-pura jadi satelit. Dia akan melakukan looping terus menerus.

Aksi: Setiap 100 milidetik, dia "nembak" (POST Request) data JSON ke URL Go-lang kamu.

Analogi: Ini kayak kamu nyuruh teman kamu nge-chat WhatsApp ke kamu "Tes 1, Tes 2, Tes 3..." tanpa henti secepat mungkin.

Golang App (The Receiver):

Tugas: Membuka pintu (Port) dan menunggu data.

Aksi: Begitu data dari Python masuk, dia tangkap, dia kumpulin (batch), lalu dia simpan ke Database secepat kilat.

Grafana (The Monitor):

Tugas: Membaca database dan menampilkan grafik.

Aksi: Grafik garis naik turun sesuai data suhu/baterai yang dikirim Python tadi.

2. Cara Test & Tahunya "It Works" (Langkah Demi Langkah)
Nanti saat kamu coding, begini urutan nge-testnya:

Tahap A: Cek Server Jalan (Pakai Postman)
Sebelum script Python (satelit palsu) dinyalakan, pastikan server Golang kamu hidup.

Jalankan Golang App: go run main.go. (Akan muncul di terminal: "Listening on port 3000").

Buka Postman.

Kirim 1 request manual.

URL: POST http://localhost:3000/telemetry

Body (JSON): {"satellite_id": "SAT-01", "temp": 50}

Indikator Sukses: Di Postman dapet status 200 OK. Di terminal Golang muncul log: "Data received".

Kesimpulan: Pintu gerbang sudah terbuka.

Tahap B: "Serangan" Otomatis (Pakai Python)
Sekarang kita ganti Postman dengan Script Python biar otomatis dan cepat.

Jalankan Script Python: python satellite_sim.py.

Indikator Sukses (Terminal):

Terminal Python akan "banjir" tulisan: Sending data... 200 OK, Sending data... 200 OK.

Terminal Golang akan "banjir" tulisan: Inserted 1000 rows, Inserted 1000 rows.

Ini membuktikan sistem kamu kuat menahan "serangan" data.

Tahap C: Visualisasi (The Real "Wow")
Ini yang kamu tunjukkan ke user/interviewer (atau screenshot untuk portofolio).

Buka Browser, masuk ke Grafana (biasanya http://localhost:3000 atau port yang diset di Docker).

Bikin Query SQL simpel di Grafana: SELECT time, cpu_temp FROM telemetry WHERE satellite_id = 'SAT-01'.

Set refresh rate ke "Every 1 second".

Indikator Sukses:

Kamu akan melihat Grafik Garis (Line Chart) yang bergerak ke kanan terus menerus secara real-time.

Kalau kamu matikan script Python-nya, grafiknya berhenti. Kalau dinyalakan lagi, grafiknya jalan lagi.

Jadi, apa yang saya butuhkan untuk mulai?
Kamu tidak perlu pusing bikin UI cantik pakai HTML/CSS. Fokusnya di "Mesin"-nya. Kamu hanya butuh 3 file utama dalam 1 folder:

docker-compose.yml (Untuk menyalakan Database TimescaleDB & Grafana secara otomatis).

main.go (Server penerima data).

simulator.py (Script pengirim data palsu).


