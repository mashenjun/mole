import argparse
import pandas as pd
import tabulate
import numpy as np
import prom_metrics_feature_score

R = "\033[0;31;40m" #RED
G = "\033[0;32;40m" # GREEN
Y = "\033[0;33;40m" # Yellow
B = "\033[0;34;40m" # Blue
N = "\033[0m" # Reset

print_columns = ["weight", "score", "target_score", "distance", "name"]
verbose_columns = ["weight", "score", "target_score", "value", "target_value", "distance", "detail", "name"]


def cal_feature_score_distance(base: pd.DataFrame, target: pd.DataFrame):
    # table1 and table2 has same row
    # different distance_function lead to different calculation logic
    base['target_score'] = target['score']
    base['target_value'] = target['value']
    base['denominator'] = base.apply(
        lambda x: max(x['score'], x['target_score']) if x['distance_function'] == 'nrm_delta' else 1,
        axis=1)
    base['distance'] = (abs(base['score'] - base['target_score'])/base['denominator']).clip(upper=1)
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
    parser.add_argument('-w', '--watermark', dest='watermark', type=float, default=0.0, help='distance lower than watermark will be skipped, default is 0.0')
    parser.add_argument('-vv', '--verbose', dest='verbose', type=bool, default=False, help='if verbose is set, show detail in result table')
    args = parser.parse_args()

    base_file = args.base
    target_file = args.target
    watermark = args.watermark
    verbose = args.verbose
    base_score = pd.read_csv(base_file)
    target_score = pd.read_csv(target_file)
    result_df = cal_feature_score_distance(base_score, target_score)
    result_df['w_distance'] = result_df["weight"] * result_df["distance"]
    if args.output is not None:
        result_df.to_csv(args.output, sep=',', index=False)
    if verbose:
        print(tabulate.tabulate(result_df[verbose_columns], headers=verbose_columns, floatfmt=".3f"))
    else:
        # print_columns = ["weight", "score", "target_score", "distance", "w_distance", "name"]
        print(tabulate.tabulate(result_df[print_columns], headers=print_columns, floatfmt=".3f"))
    # calculate the weighted sum of distance
    result_df = result_df.loc[result_df['valid'] == True]
    summary = pd.DataFrame([['total weighted distance', result_df.loc[result_df['distance'] >= watermark, 'w_distance'].sum()
                                 / result_df.loc[result_df['distance'] >= watermark, 'weight'].sum()]], columns=['summary', 'value'])
    # print(tabulate.tabulate(summary, headers=summary.columns, floatfmt=".3f", showindex=False))
    # do normalized before calculate Euclidean distance
    result_df['score_nrm'] = result_df.apply(lambda x: x['score'] / max(x['score'], x['target_score']) if x['score'] > 1 else x['score'],
        axis=1)
    result_df['target_score_nrm'] = result_df.apply(
        lambda x: x['target_score'] / max(x['score'], x['target_score']) if x['target_score'] > 1 else x['target_score'],
        axis=1)
    result_df['w_score'] = result_df['score'] * result_df['weight']
    result_df['w_target_score'] = result_df['target_score'] * result_df['weight']
    # calculate Euclidean distance
    eu_dist = np.linalg.norm(result_df['score_nrm'] - result_df['target_score_nrm'])
    summary.loc[len(summary.index)] = ['total Euclidean distance', eu_dist / (1 + eu_dist)]
    # dist = np.sqrt(np.sum((result_df['score_nrm']-result_df['target_score_nrm']) ** 2))
    cos = np.dot(result_df['w_score'], result_df['w_target_score']) / (np.linalg.norm(result_df['w_score']) * np.linalg.norm(result_df['w_target_score']))
    summary.loc[len(summary.index)] = ['total Cosine distance', 1 - cos]
    print(tabulate.tabulate(summary, headers=summary.columns, floatfmt=".3f", showindex=False))
    # print("total distance score: {0:.3f}".format(weighted_sum.sum() / result_df['weight'].sum()))
