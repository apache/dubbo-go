package global

import "testing"

func TestCloneDefaultConfig(t *testing.T) {
	t.Run("ApplicationConfig", func(t *testing.T) {
		c := DefaultApplicationConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ApplicationConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ConfigCenterConfig", func(t *testing.T) {
		c := DefaultCenterConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ConfigCenterConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ConsumerConfig", func(t *testing.T) {
		c := DefaultConsumerConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ConsumerConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("CustomConfig", func(t *testing.T) {
		c := DefaultCustomConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("CustomConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("LoggerConfig", func(t *testing.T) {
		c := DefaultLoggerConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("LoggerConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("MetadataReportConfig", func(t *testing.T) {
		c := DefaultMetadataReportConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("MetadataReportConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("MethodConfig", func(t *testing.T) {
		c := &MethodConfig{}
		clone := c.Clone()
		if clone == c {
			t.Errorf("MethodConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("MetricConfig", func(t *testing.T) {
		c := DefaultMetricsConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("MetricConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("OtelConfig", func(t *testing.T) {
		c := DefaultOtelConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("OtelConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ProfilesConfig", func(t *testing.T) {
		c := DefaultProfilesConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ProfilesConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ProtocolConfig", func(t *testing.T) {
		c := DefaultProtocolConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ProtocolConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ProviderConfig", func(t *testing.T) {
		c := DefaultProviderConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ProviderConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ReferenceConfig", func(t *testing.T) {
		c := DefaultReferenceConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ReferenceConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("RegistryConfig", func(t *testing.T) {
		c := DefaultRegistryConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("RegistryConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ServiceConfig", func(t *testing.T) {
		c := DefaultServiceConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ServiceConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("ShutdownConfig", func(t *testing.T) {
		c := DefaultShutdownConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("ShutdownConfig.Clone() = %v, want %v", clone, c)
		}
	})

	t.Run("TLSConfig", func(t *testing.T) {
		c := DefaultTLSConfig()
		clone := c.Clone()
		if clone == c {
			t.Errorf("TLSConfig.Clone() = %v, want %v", clone, c)
		}
	})
}
