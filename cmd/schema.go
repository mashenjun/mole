package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/mashenjun/mole/collector/schema"
	"github.com/mashenjun/mole/desensitize"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
)

func schemaCmd() *cobra.Command {
	// for connect to the database
	cfg := schema.MysqlConfig{}
	// for init the aes encrypt
	sk := ""
	// outPath is where store the schema sql collected
	output := ""
	// list of databases want to collect schema from
	databases := make([]string,0)
	// user := ""
	// pwd := ""
	// host := ""
	// port := ""
	cmd := &cobra.Command{
		Use:   `schema`,
		Short: `collect schema from target databases`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() != 4 {
				_ = cmd.Help()
				return errors.New("miss some flag")
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
			if err := os.MkdirAll(output, 0755); err != nil {
				fmt.Printf("make dir error: %+v\n", err)
				return err
			}
			db, err := schema.Dial(&cfg)
			if err != nil {
				fmt.Printf("dail database error: %+v\n", err)
				return err
			}

			for _, database := range databases {
				fmt.Printf("collect schema from %s\n", database)
				file, err := os.OpenFile(filepath.Join(output, fmt.Sprintf("%s.sql",database) ), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0755)
				if err != nil {
					fmt.Printf("access output file error: %+v\n", err)
					return err
				}
				sc, err := schema.NewSchemaCollector(db, database)
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
			}
			return nil
		},
	}

	// fill the mysql config form the flag
	cmd.Flags().StringSliceVar(&databases, "db", nil, "the list databases want to collect schema from")
	cmd.Flags().StringVar(&cfg.DSN, "dsn", "", "the dsn used to connect db")
	cmd.Flags().StringVar(&sk, "sk", "","secret key used to init aes encrypt")
	cmd.Flags().StringVarP(&output, "output", "o", "", "the output file used to store schema sql")
	//cmd.Flags().StringVarP(&user,"user", "u", "", "database user")
	//cmd.Flags().StringVarP(&pwd, "pwd","p","","database password")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "dsn")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "sk")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "output")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "db")
	return cmd
}