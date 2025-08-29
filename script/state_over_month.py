#!/usr/bin/env python3

import pandas as pd
import matplotlib.pyplot as plt
import numpy as np


def calculate_monthly_increases(csv_file):
    """
    Calculate monthly increases for account size, storage size, trienode size, and code size.

    Args:
        csv_file (str): Path to the CSV file containing daily state data

    Returns:
        pd.DataFrame: DataFrame containing monthly increases for each category
    """
    # Read the CSV file
    df = pd.read_csv(csv_file)

    # Convert st_date to datetime
    df["st_date"] = pd.to_datetime(df["st_date"])

    # Create year-month column
    df["year_month"] = df["st_date"].dt.to_period("M")

    # Get the last day of each month to calculate monthly values
    monthly_data = df.groupby("year_month").last().reset_index()

    # Calculate monthly increases
    monthly_increases = []

    for i in range(1, len(monthly_data)):
        current = monthly_data.iloc[i]
        previous = monthly_data.iloc[i - 1]

        account_increase = int(current["accountsize"] - previous["accountsize"])
        storage_increase = int(current["storagesize"] - previous["storagesize"])
        trienode_increase = int(current["trienodesize"] - previous["trienodesize"])
        code_increase = int(current["codesize"] - previous["codesize"])

        monthly_increases.append(
            {
                "year_month": str(current["year_month"]),
                "account_increase": account_increase,
                "storage_increase": storage_increase,
                "trienode_increase": trienode_increase,
                "code_increase": code_increase,
            }
        )

    return pd.DataFrame(monthly_increases)


def create_monthly_increase_plot(monthly_df):
    """
    Create a bar plot showing monthly increases for all categories.

    Args:
        monthly_df (pd.DataFrame): DataFrame containing monthly increase data
    """
    # Set up the plot
    _, ax = plt.subplots(figsize=(15, 8))

    # Prepare data for grouped bar chart
    x = np.arange(len(monthly_df))
    width = 0.2

    # Convert bytes to MB for plotting
    account_mb = monthly_df["account_increase"] / (1024 * 1024)
    storage_mb = monthly_df["storage_increase"] / (1024 * 1024)
    trienode_mb = monthly_df["trienode_increase"] / (1024 * 1024)
    code_mb = monthly_df["code_increase"] / (1024 * 1024)

    # Create bars for each category
    bars1 = ax.bar(
        x - 1.5 * width,
        account_mb,
        width,
        label="Account Size",
        color="green",
        alpha=0.7,
    )
    bars2 = ax.bar(
        x - 0.5 * width,
        storage_mb,
        width,
        label="Storage Size",
        color="blue",
        alpha=0.7,
    )
    bars3 = ax.bar(
        x + 0.5 * width,
        trienode_mb,
        width,
        label="Trienode Size",
        color="red",
        alpha=0.7,
    )
    bars4 = ax.bar(
        x + 1.5 * width,
        code_mb,
        width,
        label="Code Size",
        color="magenta",
        alpha=0.7,
    )

    # Customize the plot
    ax.set_title(
        "Monthly State Size Increases by Category", fontsize=16, fontweight="bold"
    )
    ax.set_xlabel("Month", fontsize=12)
    ax.set_ylabel("Size Increase (MB)", fontsize=12)
    ax.set_xticks(x)
    ax.set_xticklabels(monthly_df["year_month"], rotation=45, ha="right")
    ax.legend()
    ax.grid(True, alpha=0.3, axis="y")

    # Add value labels on top of bars
    def add_value_labels(bars):
        for bar in bars:
            height = bar.get_height()
            if height != 0:  # Only show label if there's an increase
                ax.annotate(
                    f"{height:.1f}",
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),  # 3 points vertical offset
                    textcoords="offset points",
                    ha="center",
                    va="bottom",
                    fontsize=8,
                )

    add_value_labels(bars1)
    add_value_labels(bars2)
    add_value_labels(bars3)
    add_value_labels(bars4)

    # Adjust layout and save
    plt.tight_layout()
    plt.savefig("monthly_increases.png", dpi=300, bbox_inches="tight")
    print("Monthly increases plot saved as 'monthly_increases.png'")

    # Show the plot
    plt.show()


def calculate_yearly_increases(csv_file):
    """
    Calculate yearly increases for account size, storage size, trienode size, and code size.

    Args:
        csv_file (str): Path to the CSV file containing daily state data

    Returns:
        pd.DataFrame: DataFrame containing yearly increases for each category
    """
    # Read the CSV file
    df = pd.read_csv(csv_file)

    # Convert st_date to datetime
    df["st_date"] = pd.to_datetime(df["st_date"])

    # Create year column
    df["year"] = df["st_date"].dt.year

    # Get the last day of each year to calculate yearly values
    yearly_data = df.groupby("year").last().reset_index()

    # Calculate yearly increases
    yearly_increases = []

    for i in range(len(yearly_data)):
        current = yearly_data.iloc[i]

        if i == 0:
            # For the first year, use 0 as baseline
            account_increase = current["accountsize"] / (1024 * 1024 * 1024)
            storage_increase = current["storagesize"] / (1024 * 1024 * 1024)
            trienode_increase = current["trienodesize"] / (1024 * 1024 * 1024)
            code_increase = current["codesize"] / (1024 * 1024 * 1024)
        else:
            # For subsequent years, calculate increase from previous year
            previous = yearly_data.iloc[i - 1]
            account_increase = (current["accountsize"] - previous["accountsize"]) / (
                1024 * 1024 * 1024
            )
            storage_increase = (current["storagesize"] - previous["storagesize"]) / (
                1024 * 1024 * 1024
            )
            trienode_increase = (current["trienodesize"] - previous["trienodesize"]) / (
                1024 * 1024 * 1024
            )
            code_increase = (current["codesize"] - previous["codesize"]) / (
                1024 * 1024 * 1024
            )

        # Calculate combined state size (account + storage)
        state_increase = account_increase + storage_increase
        
        yearly_increases.append(
            {
                "year": int(current["year"]),
                "account_increase_gb": account_increase,
                "storage_increase_gb": storage_increase,
                "state_increase_gb": state_increase,
                "trienode_increase_gb": trienode_increase,
                "code_increase_gb": code_increase,
            }
        )

    return pd.DataFrame(yearly_increases)


def print_yearly_increases_markdown(yearly_df):
    """
    Print yearly increases in markdown format with merged state size.

    Args:
        yearly_df (pd.DataFrame): DataFrame containing yearly increase data
    """
    print("\n# Yearly State Size Increases (GB)\n")

    # Create markdown table
    print("| Year | State Size | Trienode Size | Code Size |")
    print("|------|------------|---------------|-----------|")

    for _, row in yearly_df.iterrows():
        print(
            f"| {row['year']} | {row['state_increase_gb']:.2f} | {row['trienode_increase_gb']:.2f} | {row['code_increase_gb']:.2f} |"
        )

    print("\n## Summary Statistics\n")

    # Calculate totals
    total_state = yearly_df["state_increase_gb"].sum()
    total_trienode = yearly_df["trienode_increase_gb"].sum()
    total_code = yearly_df["code_increase_gb"].sum()

    print("| Category | Total Increase | Average/Year | Max Year Increase |")
    print("|----------|----------------|--------------|-------------------|")
    print(
        f"| **State Size** | {total_state:.2f} GB | {yearly_df['state_increase_gb'].mean():.2f} GB | {yearly_df['state_increase_gb'].max():.2f} GB |"
    )
    print(
        f"| **Trienode Size** | {total_trienode:.2f} GB | {yearly_df['trienode_increase_gb'].mean():.2f} GB | {yearly_df['trienode_increase_gb'].max():.2f} GB |"
    )
    print(
        f"| **Code Size** | {total_code:.2f} GB | {yearly_df['code_increase_gb'].mean():.2f} GB | {yearly_df['code_increase_gb'].max():.2f} GB |"
    )

    print(
        f"\n**Analysis Period:** {yearly_df['year'].min()} - {yearly_df['year'].max()} ({len(yearly_df)} years)\n"
    )


def main():
    """
    Main function to calculate monthly increases and generate visualizations.
    """
    # Calculate monthly increases
    print("Calculating monthly increases...")
    monthly_increases_df = calculate_monthly_increases("../datasets/daily-states.csv")

    # Save to CSV
    output_csv = "monthly_increases.csv"
    monthly_increases_df.to_csv(output_csv, index=False)
    print(f"Monthly increases saved to '{output_csv}'")

    # Display basic statistics
    print("\n=== Monthly Increases Summary ===")
    print(f"Total months analyzed: {len(monthly_increases_df)}")
    min_st = monthly_increases_df["year_month"].iloc[0]
    max_st = monthly_increases_df["year_month"].iloc[-1]
    print(f"Date range: {min_st} to {max_st}")

    for category in ["account", "storage", "trienode", "code"]:
        col = f"{category}_increase"
        avg_increase = monthly_increases_df[col].mean()
        max_increase = monthly_increases_df[col].max()
        max_month = monthly_increases_df.loc[
            monthly_increases_df[col].idxmax(), "year_month"
        ]
        print(f"\n{category.title()} Size:")
        print(f"  Average monthly increase: {avg_increase / (1024 * 1024):.2f} MB")
        print(
            f"  Maximum monthly increase: {max_increase / (1024 * 1024):.2f} MB (in {max_month})"
        )

    # Create visualization
    create_monthly_increase_plot(monthly_increases_df)

    # Calculate and display yearly increases
    print("\nCalculating yearly increases...")
    yearly_increases_df = calculate_yearly_increases("../datasets/daily-states.csv")
    print_yearly_increases_markdown(yearly_increases_df)


if __name__ == "__main__":
    main()
