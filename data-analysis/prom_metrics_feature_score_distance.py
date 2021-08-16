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
    result = base.sort_values(by=['distance'], ascending=False)
    return result


def cal_distance_from_basic(feature_functions_dict: dict, base_features: pd.DataFrame, target_features: pd.DataFrame):
    b_score = prom_metrics_feature_score.cal_weighted_feature_score(base_features, feature_functions_dict)
    t_score = prom_metrics_feature_score.cal_weighted_feature_score(target_features, feature_functions_dict)
    return cal_feature_score_distance(b_score, t_score)


if __name__ == '__main__':
    # we assume all the df has the weight column, and has same order

    # if len(sys.argv) != 4:
    #     raise ValueError(__file__+" <feature_functions.yaml> <base_features.csv> <target_feature.csv>")

    # feature_functions_dict = prom_metrics_feature_score.load_yaml(sys.argv[1])
    # base_features = prom_metrics_feature_score.load_feature(sys.argv[2])
    # target_features = prom_metrics_feature_score.load_feature(sys.argv[3])
    # base_score = prom_metrics_feature_score.cal_weighted_feature_score(base_features, feature_functions_dict)
    # target_score = prom_metrics_feature_score.cal_weighted_feature_score(target_features, feature_functions_dict)
    # result_df = cal_feature_score_distance(base_score, target_score)
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
    print(tabulate.tabulate(result_df, headers=result_df.columns))
    if args.output is not None:
        result_df.to_csv(args.output, sep=',', index=False)


