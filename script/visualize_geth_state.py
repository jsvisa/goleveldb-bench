#!/usr/bin/env python3

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates


def visualize_geth_state(csv_file):
    """
    Visualize account size, storage size, trienode size, and code size over time from Geth daily states CSV.

    Args:
        csv_file (str): Path to the CSV file containing daily state data
    """
    # Read the CSV file
    df = pd.read_csv(csv_file)

    # Convert st_date to datetime
    df["st_date"] = pd.to_datetime(df["st_date"])

    # Convert sizes to GB for better readability
    df["accountsize_gb"] = df["accountsize"] / (1024 * 1024 * 1024)
    df["storagesize_gb"] = df["storagesize"] / (1024 * 1024 * 1024)
    df["trienodesize_gb"] = df["trienodesize"] / (1024 * 1024 * 1024)
    df["codesize_gb"] = df["codesize"] / (1024 * 1024 * 1024)

    # Create the visualization
    _, (ax1, ax2, ax3, ax4) = plt.subplots(4, 1, figsize=(12, 20))

    # Plot 1: Account Size over time
    ax1.plot(
        df["st_date"], df["accountsize_gb"], "g-", linewidth=1.5, label="Account Size"
    )
    ax1.set_title("Account Size Over Time", fontsize=14, fontweight="bold")
    ax1.set_ylabel("Account Size (GB)", fontsize=12)
    ax1.grid(True, alpha=0.3)
    ax1.legend()
    # Add final size text
    ax1.text(
        0.02,
        0.85,
        f"Latest: {df['accountsize_gb'].iloc[-1]:.2f} GB\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax1.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightgreen", alpha=0.7),
    )

    # Format x-axis for first plot
    ax1.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax1.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 2: Storage Size over time
    ax2.plot(
        df["st_date"], df["storagesize_gb"], "b-", linewidth=1.5, label="Storage Size"
    )
    ax2.set_title("Storage Size Over Time", fontsize=14, fontweight="bold")
    ax2.set_ylabel("Storage Size (GB)", fontsize=12)
    ax2.grid(True, alpha=0.3)
    ax2.legend()
    # Add final size text
    ax2.text(
        0.02,
        0.85,
        f"Latest: {df['storagesize_gb'].iloc[-1]:.2f} GB\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax2.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightblue", alpha=0.7),
    )

    # Format x-axis for second plot
    ax2.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax2.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 3: Trienode Size over time
    ax3.plot(
        df["st_date"], df["trienodesize_gb"], "r-", linewidth=1.5, label="Trienode Size"
    )
    ax3.set_title("Trienode Size Over Time", fontsize=14, fontweight="bold")
    ax3.set_ylabel("Trienode Size (GB)", fontsize=12)
    ax3.grid(True, alpha=0.3)
    ax3.legend()
    # Add final size text
    ax3.text(
        0.02,
        0.85,
        f"Latest: {df['trienodesize_gb'].iloc[-1]:.2f} GB\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax3.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightcoral", alpha=0.7),
    )

    # Format x-axis for third plot
    ax3.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax3.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 4: Code Size over time
    ax4.plot(df["st_date"], df["codesize_gb"], "m-", linewidth=1.5, label="Code Size")
    ax4.set_title("Code Size Over Time", fontsize=14, fontweight="bold")
    ax4.set_xlabel("Date", fontsize=12)
    ax4.set_ylabel("Code Size (GB)", fontsize=12)
    ax4.grid(True, alpha=0.3)
    ax4.legend()
    # Add final size text
    ax4.text(
        0.02,
        0.85,
        f"Latest: {df['codesize_gb'].iloc[-1]:.2f} GB\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax4.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="plum", alpha=0.7),
    )

    # Format x-axis for fourth plot
    ax4.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax4.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Rotate x-axis labels for better readability
    plt.setp(ax1.xaxis.get_majorticklabels(), rotation=45)
    plt.setp(ax2.xaxis.get_majorticklabels(), rotation=45)
    plt.setp(ax3.xaxis.get_majorticklabels(), rotation=45)
    plt.setp(ax4.xaxis.get_majorticklabels(), rotation=45)

    # Adjust spacing between subplots
    plt.tight_layout(pad=3.0)

    # Save the plot as PNG
    plt.savefig("geth_state_visualization.png", dpi=300, bbox_inches="tight")
    print("Plot saved as 'geth_state_visualization.png'")

    # Show the plot
    plt.show()

    # Print some basic statistics
    print("\n=== Data Summary ===")
    min_st = df["st_date"].min().strftime("%Y-%m-%d")
    max_st = df["st_date"].max().strftime("%Y-%m-%d")
    print(f"Date range: {min_st} to {max_st}")
    print(f"Total data points: {len(df)}")
    print(f"\nAccount Size:")
    print(f"  Max: {df['accountsize_gb'].max():.2f} GB")
    print(f"  Final: {df['accountsize_gb'].iloc[-1]:.2f} GB")
    print(f"\nStorage Size:")
    print(f"  Max: {df['storagesize_gb'].max():.2f} GB")
    print(f"  Final: {df['storagesize_gb'].iloc[-1]:.2f} GB")
    print(f"\nTrienode Size:")
    print(f"  Max: {df['trienodesize_gb'].max():.2f} GB")
    print(f"  Final: {df['trienodesize_gb'].iloc[-1]:.2f} GB")
    print(f"\nCode Size:")
    print(f"  Max: {df['codesize_gb'].max():.2f} GB")
    print(f"  Final: {df['codesize_gb'].iloc[-1]:.2f} GB")


if __name__ == "__main__":
    # Run the visualization
    visualize_geth_state("../datasets/daily-states.csv")
