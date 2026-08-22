package sheets

// ThemeCorporateNavy returns the standard Alexandria theme: dark navy header with bold white text
// and subtle cool blue/grey alternating row banding.
func ThemeCorporateNavy() *Theme {
	return &Theme{
		HeaderBackground: Hex("#1C4E78"), // RGB(28, 78, 120) -> ~ 0.11, 0.31, 0.47.
		HeaderForeground: Hex("#FFFFFF"),
		HeaderBold:       true,
		EnableBanding:    true,
		ZebraFirstBand:   Hex("#FFFFFF"),
		ZebraSecondBand:  Hex("#F0F4F8"),
		AutoFitColumns:   true,
	}
}

// ThemeModernSlate returns a neutral modern aesthetic with dark slate header
// and light slate alternating rows.
func ThemeModernSlate() *Theme {
	return &Theme{
		HeaderBackground: Hex("#334155"),
		HeaderForeground: Hex("#FFFFFF"),
		HeaderBold:       true,
		EnableBanding:    true,
		ZebraFirstBand:   Hex("#FFFFFF"),
		ZebraSecondBand:  Hex("#F8FAFC"),
		AutoFitColumns:   true,
	}
}

// ThemeEmeraldForest returns a rich green accent header suited for success/financial reports.
func ThemeEmeraldForest() *Theme {
	return &Theme{
		HeaderBackground: Hex("#065F46"),
		HeaderForeground: Hex("#FFFFFF"),
		HeaderBold:       true,
		EnableBanding:    true,
		ZebraFirstBand:   Hex("#FFFFFF"),
		ZebraSecondBand:  Hex("#F0FDF4"),
		AutoFitColumns:   true,
	}
}

// ThemeCleanMinimal returns a lightweight design with soft grey headers and dark text.
func ThemeCleanMinimal() *Theme {
	return &Theme{
		HeaderBackground: Hex("#E2E8F0"),
		HeaderForeground: Hex("#0F172A"),
		HeaderBold:       true,
		EnableBanding:    false,
		AutoFitColumns:   true,
	}
}
