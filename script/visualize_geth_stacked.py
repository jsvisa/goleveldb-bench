#!/usr/bin/env python3

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.dates as mdates


def visualize_geth_stacked(csv_file):
    """
    Visualize all state size categories stacked together over time from Geth daily states CSV.

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

    # Create the stacked area plot
    _, ax = plt.subplots(figsize=(15, 10))

    # Create stacked area chart
    ax.stackplot(
        df["st_date"],
        df["accountsize_gb"],
        df["storagesize_gb"],
        df["trienodesize_gb"],
        df["codesize_gb"],
        labels=["Account Size", "Storage Size", "Trienode Size", "Code Size"],
        colors=["lightgreen", "lightblue", "lightcoral", "plum"],
        alpha=0.8,
    )

    # Customize the plot
    ax.set_title(
        "Ethereum State Size Over Time (Stacked)", fontsize=16, fontweight="bold"
    )
    ax.set_xlabel("Date", fontsize=12)
    ax.set_ylabel("Cumulative Size (GB)", fontsize=12)
    ax.legend(loc="upper center", frameon=True, fancybox=True, shadow=True)
    ax.grid(True, alpha=0.3)

    # Format x-axis
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%Y"))
    ax.xaxis.set_major_locator(mdates.YearLocator())
    ax.xaxis.set_minor_locator(mdates.MonthLocator(interval=6))

    # Rotate x-axis labels for better readability
    plt.setp(ax.xaxis.get_majorticklabels(), rotation=45)

    # Add total size annotation
    total_size = (
        df["accountsize_gb"].iloc[-1]
        + df["storagesize_gb"].iloc[-1]
        + df["trienodesize_gb"].iloc[-1]
        + df["codesize_gb"].iloc[-1]
    )
    ax.text(
        0.98,
        0.95,
        f"Total Size: {total_size:.1f} GB\n({df['st_date'].iloc[-1].strftime('%Y-%m-%d')})",
        transform=ax.transAxes,
        fontsize=12,
        ha="right",
        va="top",
        bbox=dict(
            boxstyle="round,pad=0.5", facecolor="white", alpha=0.9, edgecolor="gray"
        ),
    )

    # Add individual category values at the end
    latest_values = [
        df["accountsize_gb"].iloc[-1],
        df["storagesize_gb"].iloc[-1],
        df["trienodesize_gb"].iloc[-1],
        df["codesize_gb"].iloc[-1],
    ]
    categories = ["Account", "Storage", "Trienode", "Code"]

    info_text = "Latest Sizes:\n" + "\n".join(
        [f"{cat}: {val:.1f} GB" for cat, val in zip(categories, latest_values)]
    )
    ax.text(
        0.02,
        0.98,
        info_text,
        transform=ax.transAxes,
        fontsize=10,
        ha="left",
        va="top",
        bbox=dict(
            boxstyle="round,pad=0.5",
            facecolor="lightyellow",
            alpha=0.9,
            edgecolor="gray",
        ),
    )

    # Adjust layout
    plt.tight_layout()

    # Save the plot as PNG
    plt.savefig("geth_stacked_visualization.png", dpi=300, bbox_inches="tight")
    print("Stacked plot saved as 'geth_stacked_visualization.png'")

    # Show the plot
    plt.show()

    # Print summary statistics
    print("\n=== Stacked Size Data Summary ===")
    min_st = df["st_date"].min().strftime("%Y-%m-%d")
    max_st = df["st_date"].max().strftime("%Y-%m-%d")
    print(f"Total data points: {len(df)}")
    print(f"\nFinal Sizes (GB):")
    print(
        f"  Account Size: {df['accountsize_gb'].iloc[-1]:.2f} GB ({df['accountsize_gb'].iloc[-1]/total_size*100:.1f}%)"
    )
    print(
        f"  Storage Size: {df['storagesize_gb'].iloc[-1]:.2f} GB ({df['storagesize_gb'].iloc[-1]/total_size*100:.1f}%)"
    )
    print(
        f"  Trienode Size: {df['trienodesize_gb'].iloc[-1]:.2f} GB ({df['trienodesize_gb'].iloc[-1]/total_size*100:.1f}%)"
    )
    print(
        f"  Code Size: {df['codesize_gb'].iloc[-1]:.2f} GB ({df['codesize_gb'].iloc[-1]/total_size*100:.1f}%)"
    )
    print(f"  Total Size: {total_size:.2f} GB")

    # Growth analysis
    initial_total = (
        df["accountsize_gb"].iloc[0]
        + df["storagesize_gb"].iloc[0]
        + df["trienodesize_gb"].iloc[0]
        + df["codesize_gb"].iloc[0]
    )
    total_growth = total_size - initial_total
    print(f"\nGrowth Analysis:")
    print(f"  Initial Total: {initial_total:.2f} GB")
    print(f"  Total Growth: {total_growth:.2f} GB")
    print(f"  Growth Factor: {total_size/initial_total:.1f}x")


if __name__ == "__main__":
    # Run the visualization
    visualize_geth_stacked("../datasets/daily-states.csv")
