# go-restaurant-managament
# Restaurant Management System (RMS)

**Technologies:** Golang (Gin), MongoDB (Aggregation), Vue.js, WebSocket, JWT, Docker  
**Live Demo:** [http://13.51.198.170:9000/frontend](http://13.51.198.170:9000/frontend)  
*For detailed usage, login credentials, and role-based access, see below.*

## Project Overview
RMS is a full-stack application designed to streamline restaurant operations, including order management, kitchen notifications, and invoicing.

## Features
- **Order Management:** Allows waitstaff to create orders and notify the kitchen.
- **Real-Time Notifications:** WebSocket notifications for order updates with sound alerts.
- **Role-Based Access:** Admin, Waiter, and Kitchen roles with tailored permissions.
- **Sales Reporting:** MongoDB aggregation to generate real-time reports.

## GitHub Repositories

- **Backend (Golang):** [go-restaurant-management](https://github.com/mamonthee/go-restaurant-managament)
- **Frontend (Vue.js):** [restaurant-management-frontend](https://github.com/mamonthee/restaurant-management-frontend)

## Login Instructions

**Demo Credentials:**

- **Admin Role**
  - **Username:** `admin@example.com`
  - **Password:** `admin123`
- **Waiter Role**
  - **Username:** `waiter@example.com`
  - **Password:** `waiter123`
- **Kitchen Role**
  - **Username:** `kitchen@example.com`
  - **Password:** `kitchen123`

## Usage Guide

- **Admin Dashboard:** Manage users, menus, and access sales reports.
- **Order Placement (Waiter):** Create and track orders.
- **Order Processing (Kitchen):** Update order status to notify waitstaff.

## Technologies Used
- **Backend:** Golang (Gin)
- **Frontend:** Vue.js
- **Database:** MongoDB (Aggregation)
- **Real-Time:** WebSocket for notifications
- **Authentication:** JWT for secure access

## Installation and Setup
To set up the project locally:

### Backend Setup:
1. Clone the backend repository:
   ```bash
   git clone https://github.com/mamonthee/go-restaurant-managament.git
