package main

import (
	"context"
	"fmt"
	"github.com/mashenjun/mole/collector"
	"github.com/mashenjun/mole/desensitize"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
)

func schemaCmd() *cobra.Command {
	// for connect to the database
	cfg := collector.MysqlConfig{}
	// for init the aes encrypt
	sk := ""
	// outPath is where store the schema sql collected
	output := ""
	cmd := &cobra.Command{
		Use:   `schema`,
		Short: `collect schema from target database`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() != 3 {
				return cmd.Help()
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: init the related component and start the process
			enc, err := desensitize.NewAESEncrypt([]byte(sk))
			if err != nil {
				fmt.Printf("new aes encrypt error: %+v\n", err)
				return err
			}
			// in append mode
			file, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				fmt.Printf("access output file error: %+v\n", err)
				return err
			}
			db, err := collector.Dial(&cfg)
			if err != nil {
				fmt.Printf("dail database error: %+v\n", err)
				return err
			}
			sc, err := collector.NewSchemaCollector(db)
			if err != nil {
				fmt.Printf("new schema collector error: %+v\n", err)
				return err
			}
			sm , err := desensitize.NewSchemaMask(enc, sc.GetSink(), file)
			if err != nil {
				fmt.Printf("new schema mask error: %+v\n", err)
				return err
			}
			errG, ctx := errgroup.WithContext(context.Background())
			errG.Go(sc.Collect)
			errG.Go(func() error {
				return sm.Start(ctx)
			})
			if err := errG.Wait(); err != nil {
				fmt.Printf("collect schema error: %+v\n", err)
				return err
			}
			return nil
		},
	}

	// fill the mysql config form the flag
	cmd.Flags().StringVarP(&cfg.DSN, "dsn", "d", "", "the dsn used to connect db")
	cmd.Flags().StringVar(&sk, "sk", "","secret key used to init aes encrypt")
	cmd.Flags().StringVarP(&output, "output", "o", "", "the out file used to store schema sql")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "dsn")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "sk")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "output")
	return cmd
}