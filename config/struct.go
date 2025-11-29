package config

type Config struct {
	// General configuration
	Env string `yaml:"env" mapstructure:"env" validate:"required"`
	Log Log    `yaml:"log" mapstructure:"log" validate:"required"`
	App App    `yaml:"app" mapstructure:"app" validate:"required"`

	// Infrastructure components
	Objectstore Objectstore `yaml:"objectstore" mapstructure:"objectstore" validate:"required"`
}

type App struct {
	Name      string `yaml:"name" mapstructure:"name" validate:"required"`
	JobBuffer int    `yaml:"jobBuffer" mapstructure:"jobBuffer" validate:"required,gte=1"`
	JWT       JWT    `yaml:"jwt" mapstructure:"jwt" validate:"required"`
}

type JWT struct {
	Secret               string `yaml:"secret" mapstructure:"secret" validate:"required"`
	AccessTokenDuration  int64  `yaml:"accessTokenDuration" mapstructure:"accessTokenDuration" validate:"required,gte=1"`
	RefreshTokenDuration int64  `yaml:"refreshTokenDuration" mapstructure:"refreshTokenDuration" validate:"required,gte=1"`
	RefreshSecret        string `yaml:"refreshSecret" mapstructure:"refreshSecret"`
}

type Log struct {
	Level      string `yaml:"level" mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format     string `yaml:"format" mapstructure:"format" validate:"oneof=json text"`
	AddSource  bool   `yaml:"addSource" mapstructure:"addSource" validate:"required"`
	TimeFormat string `yaml:"timeFormat" mapstructure:"timeFormat" validate:"required"`
}

type Objectstore struct {
	PresignedDefaultTTL int64            `yaml:"presignedDefaultTTL" mapstructure:"presignedDefaultTTL" validate:"gte=1"`
	Local               LocalObjectstore `yaml:"local" mapstructure:"local"`
	Storj               StorjObjectstore `yaml:"storj" mapstructure:"storj"`
	Cache               CacheObjectstore `yaml:"cache" mapstructure:"cache"`
}

type LocalObjectstore struct {
	Root    string `yaml:"root" mapstructure:"root" validate:"required"`
	BaseURL string `yaml:"baseUrl" mapstructure:"baseUrl" validate:"required,url"`
}

type StorjObjectstore struct {
	Bucket      string `yaml:"bucket" mapstructure:"bucket" validate:"required"`
	AccessGrant string `yaml:"accessGrant" mapstructure:"accessGrant" validate:"required"`
	BaseURL     string `yaml:"baseUrl" mapstructure:"baseUrl" validate:"required,url"`
}

type CacheObjectstore struct {
	MaxSize int64 `yaml:"maxSize" mapstructure:"maxSize" validate:"required,gte=1"`
}
