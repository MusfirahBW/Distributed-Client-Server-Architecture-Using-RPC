# Distributed Matrix Computation with RPC

## 📌 Description

This project implements a **distributed client-server architecture** for performing matrix operations using **Remote Procedure Calls (RPC)**. The system consists of:

- **Client**: Sends computation requests.
- **Coordinator (Server)**: Manages task distribution among workers.
- **Workers**: Perform matrix computations.

The coordinator ensures **task scheduling (FCFS)**, **load balancing**, and **fault tolerance**. Additionally, a **TLS connection** is implemented for secure communication.

---

## 📂 Project Structure  
- **`client.go`** – Sends computation requests to the coordinator via RPC and receives results.  
- **`coordinator.go`** – Acts as the central server, distributing tasks among workers and handling failures.  
- **`worker/`** – Directory containing worker implementation files.  
- **`server.crt`** – Security certificate for TLS communication.  
- **`server.key`** – Private key for TLS encryption.  

---

## 🚀 Features

### ✅ Client

- Sends matrix computation requests to the coordinator over RPC.
- Receives computed results and displays them.
- Communicates securely using **TLS encryption**.

### ✅ Coordinator (Server)

- Schedules tasks using **First-Come, First-Served (FCFS)**.
- Assigns tasks to **least busy workers** (Load Balancing).
- Implements **fault tolerance**: reassigns failed tasks to another worker.
- Sends computed results back to the client.

### ✅ Worker

- Executes matrix operations: **addition, transpose, multiplication**.
- Receives tasks from the coordinator and returns results.
- Reports failures if encountered.

### ✅ Bonus: Secure Communication

- Implements **TLS encryption** for secure RPC communication.
- Uses **SSL certificates** to authenticate the client and server.

---

## ⚡ How It Works

1. The **client** sends a matrix operation request to the **coordinator**.
2. The **coordinator** assigns the task to the least busy **worker**.
3. The **worker** performs the requested computation and sends the result back.
4. If a **worker fails**, the coordinator reassigns the task.
5. The **client** receives and displays the result.

---

## 🛠️ Setup & Run

### Prerequisites

- **Go 1.19+**
- **OpenSSL** (for generating TLS certificates)

### Steps

1. First run the coordinator, then the worker and lastly the client.
2. Commands to run each file in a separate terminal are written at the end of each respective code file.
3. The worker and coordinator are working on the same device and the client is on a different physical device and for the code to work they must be connected to the same internet.
4. When we generate the certificate for TLS in a coordinator terminal, we need to copy the security certificate and key into the worker directory as well. Not only this, we need to share the security certificate with the client too.
---



