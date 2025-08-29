#!/usr/bin/env python3

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates


def visualize_geth_counts(csv_file):
    """
    Visualize account, storage, trienode, and code counts over time from Geth daily states CSV.

    Args:
        csv_file (str): Path to the CSV file containing daily state data
    """
    # Read the CSV file
    df = pd.read_csv(csv_file)

    # Convert st_date to datetime
    df["st_date"] = pd.to_datetime(df["st_date"])

    # Create the visualization
    _, (ax1, ax2, ax3, ax4) = plt.subplots(4, 1, figsize=(12, 20))

    # Plot 1: Account Count over time
    ax1.plot(
        df["st_date"],
        df["accounts"] / 1_000_000,
        "g-",
        linewidth=1.5,
        label="Account Count",
    )
    ax1.set_title("Account Count Over Time", fontsize=14, fontweight="bold")
    ax1.set_ylabel("Count (millions)", fontsize=12)
    ax1.grid(True, alpha=0.3)
    ax1.legend()
    # Add final count text
    ax1.text(
        0.02,
        0.85,
        f"Latest: {df['accounts'].iloc[-1]:,}\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax1.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightgreen", alpha=0.7),
    )

    # Format x-axis for first plot
    ax1.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax1.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 2: Storage Count over time
    ax2.plot(
        df["st_date"],
        df["storages"] / 1_000_000_000,
        "b-",
        linewidth=1.5,
        label="Storage Count",
    )
    ax2.set_title("Storage Count Over Time", fontsize=14, fontweight="bold")
    ax2.set_ylabel("Count (billions)", fontsize=12)
    ax2.grid(True, alpha=0.3)
    ax2.legend()
    # Add final count text
    ax2.text(
        0.02,
        0.85,
        f"Latest: {df['storages'].iloc[-1]:,}\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax2.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightblue", alpha=0.7),
    )

    # Format x-axis for second plot
    ax2.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax2.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 3: Trienode Count over time
    ax3.plot(
        df["st_date"],
        df["trienodes"] / 1_000_000_000,
        "r-",
        linewidth=1.5,
        label="Trienode Count",
    )
    ax3.set_title("Trienode Count Over Time", fontsize=14, fontweight="bold")
    ax3.set_ylabel("Count (billions)", fontsize=12)
    ax3.grid(True, alpha=0.3)
    ax3.legend()
    # Add final count text
    ax3.text(
        0.02,
        0.85,
        f"Latest: {df['trienodes'].iloc[-1]:,}\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax3.transAxes,
        fontsize=10,
        bbox=dict(boxstyle="round,pad=0.3", facecolor="lightcoral", alpha=0.7),
    )

    # Format x-axis for third plot
    ax3.xaxis.set_major_formatter(mdates.DateFormatter("%Y-%m"))
    ax3.xaxis.set_major_locator(mdates.MonthLocator(interval=6))

    # Plot 4: Code Count over time
    ax4.plot(
        df["st_date"], df["codes"] / 1_000_000, "m-", linewidth=1.5, label="Code Count"
    )
    ax4.set_title("Code Count Over Time", fontsize=14, fontweight="bold")
    ax4.set_xlabel("Date", fontsize=12)
    ax4.set_ylabel("Count (millions)", fontsize=12)
    ax4.grid(True, alpha=0.3)
    ax4.legend()
    # Add final count text
    ax4.text(
        0.02,
        0.85,
        f"Latest: {df['codes'].iloc[-1]:,}\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
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
    plt.savefig("geth_count_visualization.png", dpi=300, bbox_inches="tight")
    print("Plot saved as 'geth_count_visualization.png'")

    # Show the plot
    plt.show()

    # Print some basic statistics
    print("\n=== Count Data Summary ===")
    min_st = df["st_date"].min().strftime("%Y-%m-%d")
    max_st = df["st_date"].max().strftime("%Y-%m-%d")
    print(f"Date range: {min_st} to {max_st}")
    print(f"Total data points: {len(df)}")
    print(f"\nAccount Count:")
    print(f"  Max: {df['accounts'].max():,}")
    print(f"  Final: {df['accounts'].iloc[-1]:,}")
    print(f"\nStorage Count:")
    print(f"  Max: {df['storages'].max():,}")
    print(f"  Final: {df['storages'].iloc[-1]:,}")
    print(f"\nTrienode Count:")
    print(f"  Max: {df['trienodes'].max():,}")
    print(f"  Final: {df['trienodes'].iloc[-1]:,}")
    print(f"\nCode Count:")
    print(f"  Max: {df['codes'].max():,}")
    print(f"  Final: {df['codes'].iloc[-1]:,}")


if __name__ == "__main__":
    # Run the visualization
    visualize_geth_counts("../datasets/daily-states.csv")
