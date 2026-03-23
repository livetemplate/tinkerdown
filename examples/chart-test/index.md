---
title: "Chart Rendering Test"
charts:
  sales-by-region:
    colors: ["#e63946", "#457b9d", "#2a9d8f", "#e9c46a"]
    horizontal: true
  monthly-trend:
    stacked: true
  market-share:
    colors: ["#264653", "#2a9d8f", "#e9c46a"]
    legend: false
---

# Chart Rendering

This page tests Chart.js chart rendering from markdown tables.

## Sales by Region {chart:bar}

| Region | Sales |
|--------|-------|
| North  | 100   |
| South  | 150   |
| East   | 120   |
| West   | 90    |

## Monthly Trend {chart:line}

| Month | Revenue | Expenses |
|-------|---------|----------|
| Jan   | 1000    | 800      |
| Feb   | 1200    | 850      |
| Mar   | 1100    | 900      |
| Apr   | 1400    | 950      |
| May   | 1300    | 880      |
| Jun   | 1500    | 920      |

## Market Share {chart:pie}

| Product | Share |
|---------|-------|
| Widget  | 45    |
| Gadget  | 30    |
| Thing   | 25    |

## Auto-Detected Chart {chart}

| Category | Value |
|----------|-------|
| A        | 10    |
| B        | 20    |
| C        | 15    |
| D        | 25    |
| E        | 18    |
