import argparse
import os
from pathlib import Path
import sys

import pandas as pd
# cal distance between table heatmap by area ratio
import tabulate


def cal_heatmap_distance(base: pd.DataFrame, target: pd.DataFrame, name:str):
    # normalize
    if len(base.index) > 1:
        base_denominator = base.max(axis=0) - base.min(axis=0)
        base_denominator = base_denominator.apply(lambda x: 1 if x == 0 else x)
        base = (base - base.min()) / base_denominator
    if len(target.index) > 1:
        target_denominator = target.max(axis=0) - target.min(axis=0)
        target_denominator = target_denominator.apply(lambda x: 1 if x == 0 else x)
        target = (target - target.min()) / target_denominator
    base_sum: pd.Series = base.sum(axis=0)
    target_sum: pd.Series = target.sum(axis=0)
    # scale the base_sum and target_sum if they have different column counts
    base_sum.reset_index(inplace=True, drop=True)
    target_sum.reset_index(inplace=True, drop=True)
    base_sum, target_sum = align(base_sum, target_sum)
    score = cal_distance(base_sum, target_sum)
    return pd.DataFrame([[name, score]], columns=['name', 'score'])


def align(base: pd.Series, target: pd.Series):
    base_cols = base.size
    target_cols = target.size
    if base_cols == target_cols:
        return base, target
    return scale_out(base, target_cols), scale_out(target, base_cols)


def scale_out(s: pd.Series, factor: int):
    out = pd.Series([], dtype='float64')
    for i in range(0, s.size):
        elements = []
        for j in range(0, factor):
            elements.append(s[i])
        out = out.append(pd.Series(elements), ignore_index=True)
    return out


def cal_distance(base: pd.Series, target: pd.Series):
    # normalize
    diff_area = 0
    total_area = 0
    length = base.size
    for i in range(0, length):
        base_val = base.iloc[i]
        target_val = target.iloc[i]
        diff_area += abs(base.iloc[i] - target.iloc[i])
        total_area += max(base_val, target_val)
    if total_area == 0:
        return 0
    return diff_area / total_area


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description="""
            heatmap_feature_distance.py calculate distance between heatmap of base and target""")
    parser.add_argument('-b', '--base', dest='base', help='dir contain heatmap csv file',
                        required=True)
    parser.add_argument('-t', '--target', dest='target', help='dir contain heatmap csv file',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file stores the heatmap distance')
    args = parser.parse_args()

    base_heatmap_dir = args.base
    target_heatmap_dir = args.target
    base_files = os.listdir(base_heatmap_dir)
    target_files = os.listdir(target_heatmap_dir)
    result_df = pd.DataFrame(columns=['name', 'score'])
    for file in base_files:
        if file not in target_files:
            continue
        base_df = pd.read_csv(os.path.join(base_heatmap_dir, file), header=None)
        target_df = pd.read_csv(os.path.join(target_heatmap_dir, file), header=None)
        df = cal_heatmap_distance(base_df.iloc[:, 1:], target_df.iloc[:, 1:], Path(file).stem)
        result_df = result_df.append(df, ignore_index=True)
    result_df.sort_values(by=['score'], ascending=False, inplace=True, ignore_index=True)
    if args.output is not None:
        result_df.to_csv(args.output, sep=',', index=False)

    print(tabulate.tabulate(result_df, headers=result_df.columns))

