import sys
import argparse

import pandas
import pandas as pd
import tabulate

import prom_metrics_feature_score


def cal_feature_score_distance(base: pd.DataFrame, target: pd.DataFrame):
    # table1 and table2 has same row
    base['target_score'] = target['score']
    base['distance'] = abs(base['score'] - base['target_score']).clip(upper=1)
    base['distance'] = base['distance'] * base['weight']
    result = base.sort_values(by=['distance'], ascending=False, ignore_index=True)
    return result


def cal_distance_from_basic(feature_functions_dict: dict, base_features: pd.DataFrame, target_features: pd.DataFrame):
    b_score = prom_metrics_feature_score.cal_weighted_feature_score(base_features, feature_functions_dict)
    t_score = prom_metrics_feature_score.cal_weighted_feature_score(target_features, feature_functions_dict)
    return cal_feature_score_distance(b_score, t_score)


if __name__ == '__main__':
    # we assume all the df has the weight column, and has the same order
    parser = argparse.ArgumentParser(description="""
            prom_metrics_feature_score_distance.py calculate distance of feature score between base and target""")
    parser.add_argument('-b', '--base', dest='base', help='csv file contains feature score',
                        required=True)
    parser.add_argument('-t', '--target', dest='target', help='csv file contains feature score',
                        required=True)
    parser.add_argument('-o', '--output', dest='output', help='output file stores the distance between feature score')
    args = parser.parse_args()

    base_file = args.base
    target_file = args.target
    base_score = pandas.read_csv(base_file)
    target_score = pandas.read_csv(target_file)
    result_df = cal_feature_score_distance(base_score, target_score)
    if args.output is not None:
        result_df.to_csv(args.output, sep=',', index=False)
    print_columns = ["weight", "score", "target_score", "distance", "name"]
    print(tabulate.tabulate(result_df[print_columns], headers=print_columns, floatfmt=".3f"))
    # calculate the weighted sum of distance
    weighted_sum = result_df["weight"] * result_df["distance"]
    summary = pandas.DataFrame([['total distance score', weighted_sum.sum() / result_df['weight'].sum()]],
                               columns=['summary', 'value'])
    print(tabulate.tabulate(summary, headers=summary.columns, floatfmt=".3f", showindex=False))
    # print("total distance score: {0:.3f}".format(weighted_sum.sum() / result_df['weight'].sum()))