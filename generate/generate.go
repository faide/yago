package generate

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TypeConf contains column type and empty value of a go type
type TypeConf struct {
	ColumnName string
	EmptyValue string
}

// TypesMap associate column types and empty values to go types.
// It is used to guess the column type at generation time.
var TypesMap = map[string]TypeConf{
	"int":           {"qb.Int()", "0"},
	"uint":          {"qb.Int().Unsigned()", "0"},
	"int64":         {"qb.BigInt()", "0"},
	"uint64":        {"qb.BigInt().Unsigned()", "0"},
	"string":        {"qb.Varchar()", `""`},
	"*string":       {"qb.Varchar()", "nil"},
	"bool":          {"qb.Boolean()", "false"},
	"time.Time":     {"qb.Timestamp()", "(time.Time{})"},
	"*time.Time":    {"qb.Timestamp()", "nil"},
	"uuid.UUID":     {"qb.UUID()", "(uuid.UUID{})"},
	"uuid.NullUUID": {"qb.UUID()", "(uuid.NullUUID{})"},
}

func guessColumnType(goType string) (string, error) {
	typeConf, ok := TypesMap[goType]
	if ok {
		return typeConf.ColumnName, nil
	}
	return "", fmt.Errorf("Cannot guess column type for go type %s", goType)
}

func makeColumnName(name string) string {
	return ToDBName(name)
}

func getEmptyValue(goType string) (string, error) {
	typeConf, ok := TypesMap[goType]
	if ok {
		return typeConf.EmptyValue, nil
	}
	return "", fmt.Errorf("Unknown empty value for type '%v'", goType)
}

func prepareFieldData(str *StructData, f *FieldData) {
	var err error
	if f.ColumnName == "" {
		f.ColumnName = makeColumnName(f.Name)
	}
	if f.ColumnType == "" {
		f.ColumnType, err = guessColumnType(f.Type)
		if err != nil {
			panic(fmt.Sprintf("Failure on field '%s': Got err '%s'",
				f.Name, err))
		}
	}
	if f.ColumnModifiers == "" {
		if f.Tags.PrimaryKey {
			f.ColumnModifiers += ".PrimaryKey()"
		}
		if f.Tags.AutoIncrement {
			f.ColumnModifiers += ".AutoIncrement()"
		}
		if f.Tags.Null {
			f.ColumnModifiers += ".Null()"
		} else if f.Tags.NotNull {
			f.ColumnModifiers += ".NotNull()"
		} else if f.Type[0] == '*' || strings.Contains(f.Type, "Null") {
			f.ColumnModifiers += ".Null()"
		} else {
			f.ColumnModifiers += ".NotNull()"
		}
	}
	if f.EmptyValue == "" && f.Tags.PrimaryKey {
		f.EmptyValue, err = getEmptyValue(f.Type)
		if err != nil {
			panic(fmt.Sprintf("Failure on field '%s': Got err '%s'",
				f.Name, err))
		}
	}
	if f.NameConst == "" {
		f.NameConst = fmt.Sprintf(
			"%s%s",
			str.Name, f.Name,
		)
	}
	if f.ColumnNameConst == "" {
		f.ColumnNameConst = fmt.Sprintf(
			"%s%sColumnName",
			str.Name, f.Name,
		)
	}
}

func prepareStructData(str *StructData, fd FileData) {
	str.File = fd
	str.PrivateBasename = strings.ToLower(str.Name[0:1]) + str.Name[1:]
	for i := range str.Fields {
		prepareFieldData(str, &str.Fields[i])
	}
}

func loadEmbedded(path string, structs map[string]*StructData, fd FileData) []*StructData {
	var newStructs []*StructData

	otherStructsByName := make(map[string]*StructData)

	var err error
	newStructs, err = ParseDir(path)
	if err != nil {
		panic(err)
	}
	for _, str := range newStructs {
		prepareStructData(str, fd)
		otherStructsByName[str.Name] = str
	}

	allStructs := []*StructData{}
	for _, str := range structs {
		allStructs = append(allStructs, str)
	}
	allStructs = append(allStructs, newStructs...)

	for _, str := range allStructs {
		for _, name := range str.Embed {
			embedded, ok := structs[name]
			if !ok {
				embedded, ok = otherStructsByName[name]
			}
			if ok {
				for index, fields := range embedded.Indexes {
					if _, ok := str.Indexes[index]; !ok {
						str.Indexes[index] = []int{}
					}
					for _, fieldIndex := range fields {
						str.Indexes[index] = append(str.Indexes[index], len(str.Fields)+fieldIndex)
					}
				}
				for _, field := range embedded.Fields {
					field.FromEmbedded = true
					str.Fields = append(str.Fields, field)
				}
			} else {
				fmt.Println(
					"Could not find embedded struct definition for '" + name + "'")
			}
		}
	}
	return newStructs
}

func parseFkDef(fkDef string) (fk string, onUpdate string, onDelete string) {
	if strings.Index(fkDef, " ") != -1 {
		tokens := strings.Split(fkDef, " ")
		fk = tokens[0]
		for i := 1; i < len(tokens); {
			token := tokens[i]
			var event *string
			switch strings.ToUpper(token) {
			case "ONUPDATE":
				event = &onUpdate
			case "ONDELETE":
				event = &onDelete
			default:
				panic(fmt.Sprintf("Invalid token in fk definition: %s", token))
			}
			i++
			for i < len(tokens) {
				token := tokens[i]
				if strings.ToUpper(token) == "ONUPDATE" || strings.ToUpper(token) == "ONDELETE" {
					break
				}
				if *event != "" {
					*event += " "
				}
				*event += token
				i++
			}
			*event = strings.ToUpper(*event)
		}
	} else {
		fk = fkDef
	}
	return
}

func postPrepare(filedata *FileData, structs map[string]*StructData) {
	for _, str := range structs {
		for i := range str.Fields {
			if str.Fields[i].Tags.PrimaryKey {
				str.PKeyFields = append(str.PKeyFields, &str.Fields[i])
			}
			if str.Fields[i].Tags.AutoIncrement {
				str.AutoIncrementPKey = &str.Fields[i]
			}
		}
		if len(str.PKeyFields) == 0 {
			panic(fmt.Sprintf("No Primary Key found on %s", str.Name))
		}
	}
	for _, str := range structs {
		if str.Imported {
			continue
		}
		for i, f := range str.Fields {
			if f.Tags.PrimaryKey && str.Fields[i].Type == "uuid.UUID" {
				filedata.Imports["github.com/m4rw3r/uuid"] = true
			}
			for _, fkDef := range str.Fields[i].Tags.ForeignKeys {
				var (
					structName   string
					refFieldName string
					refStruct    *StructData
					refField     *FieldData
				)
				fk, onUpdate, onDelete := parseFkDef(fkDef)
				if strings.Index(fk, ".") != -1 {
					splitted := strings.Split(fk, ".")
					structName = splitted[0]
					refFieldName = splitted[1]
				} else {
					structName = fk
				}
				refStruct = structs[structName]
				if refFieldName == "" {
					refField = refStruct.PKeyFields[0]
				} else {
					for i := range refStruct.Fields {
						if refStruct.Fields[i].Name == refFieldName {
							refField = &refStruct.Fields[i]
						}
					}
				}

				str.ForeignKeys = append(str.ForeignKeys, FKData{
					Column:    &str.Fields[i],
					RefTable:  refStruct,
					RefColumn: refField,
					OnUpdate:  onUpdate,
					OnDelete:  onDelete,
				})
			}
		}
	}
}

// ProcessFile processes a go file and generates mapper and mappedstruct
// interfaces implementations for the yago structs.
func ProcessFile(logger *log.Logger, path string, file string, pack string, output string, format bool) error {

	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)

	if output == "" {
		output = filepath.Join(path, base+"_yago"+ext)
	}

	filedata := FileData{Package: pack, Imports: make(map[string]bool)}

	structs, err := ParseFile(filepath.Join(path, file))
	if err != nil {
		return err
	}

	structsByName := make(map[string]*StructData)
	for _, str := range structs {
		prepareStructData(str, filedata)
		structsByName[str.Name] = str
		if !str.NoTable {
			filedata.HasTables = true
		}
	}
	otherStructs := loadEmbedded(path, structsByName, filedata)
	for _, str := range otherStructs {
		if _, ok := structsByName[str.Name]; !ok {
			str.Imported = true
			structsByName[str.Name] = str
		}
	}
	postPrepare(&filedata, structsByName)

	outf, err := os.Create(output)
	if err != nil {
		return err
	}

	{
		defer outf.Close()

		if err := prologTemplate.Execute(outf, &filedata); err != nil {
			return err
		}

		for _, str := range structs {
			if err := structPreambleTemplate.Execute(outf, &str); err != nil {
				return err
			}
			if str.NoTable {
				continue
			}
			if err := structTemplate.Execute(outf, &str); err != nil {
				return err
			}
		}
	}

	if format {
		cmd := exec.Command("gofmt", "-s", "-w", output)
		if err := cmd.Run(); err != nil {
			logger.Fatal(err)
		}
	}

	return nil
}
