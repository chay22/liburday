package main

import (
	"strings"
	"testing"
)

const sampleKalenderkuHTML = `
<!DOCTYPE html>
<html>
<body>
<div>
	<h2>Kalender Tahun 2027 Lengkap</h2>
	<h2>Daftar Hari Libur Panjang Tahun 2027</h2>
	<ul>
		<li>1 - 3 Januari 2027 Libur panjang</li>
	</ul>
	<h2>Daftar Hari Libur Tahun 2027</h2>
	<div>
		<ul class="columns-1 lg:columns-2">
			<li>
				<button type="button">
					<span class="sr-only">Jumat 1 Januari - Tahun Baru 2027 Masehi - Libur Nasional</span>
					<div class="flex items-baseline gap-2" aria-hidden="true">
						<span class="text-sm font-medium">1 Januari</span>
					</div>
					<span class="truncate text-sm" aria-hidden="true">Tahun Baru 2027 Masehi</span>
					<span class="flex items-center gap-1.5" aria-hidden="true">
						<span class="h-1.5 w-1.5 rounded-full bg-holiday"></span>
						<span class="text-xs">Libur</span>
					</span>
				</button>
			</li>
			<li>
				<button type="button">
					<span class="sr-only">Jumat 5 Februari - Cuti Tahun Baru Imlek 2578 Kongzili - Cuti Bersama</span>
					<div class="flex items-baseline gap-2" aria-hidden="true">
						<span class="text-sm font-medium">5 Februari</span>
					</div>
					<span class="truncate text-sm" aria-hidden="true">Cuti Tahun Baru Imlek 2578 Kongzili</span>
					<span class="flex items-center gap-1.5" aria-hidden="true">
						<span class="h-1.5 w-1.5 rounded-full bg-cuti"></span>
						<span class="text-xs">Cuti</span>
					</span>
				</button>
			</li>
			<li>
				<button type="button">
					<!-- Test fallback without sr-only -->
					<div class="flex items-baseline gap-2" aria-hidden="true">
						<span class="text-sm font-medium">25 Desember</span>
					</div>
					<span class="truncate text-sm" aria-hidden="true">Hari Raya Natal</span>
				</button>
			</li>
		</ul>
	</div>
</div>
</body>
</html>
`

func TestParseKalenderkuHTML(t *testing.T) {
	holidays, err := parseKalenderkuHTML(strings.NewReader(sampleKalenderkuHTML), 2027)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(holidays) != 3 {
		t.Fatalf("expected 3 holidays, got %d", len(holidays))
	}

	expected := []Holiday{
		{Date: "2027-01-01", Name: "Tahun Baru 2027 Masehi", IsNational: 1},
		{Date: "2027-02-05", Name: "Cuti Tahun Baru Imlek 2578 Kongzili", IsNational: 1},
		{Date: "2027-12-25", Name: "Hari Raya Natal", IsNational: 1},
	}

	for i, exp := range expected {
		if holidays[i].Date != exp.Date {
			t.Errorf("[%d] expected date %s, got %s", i, exp.Date, holidays[i].Date)
		}
		if holidays[i].Name != exp.Name {
			t.Errorf("[%d] expected name %s, got %s", i, exp.Name, holidays[i].Name)
		}
		if holidays[i].IsNational != exp.IsNational {
			t.Errorf("[%d] expected is_national %d, got %d", i, exp.IsNational, holidays[i].IsNational)
		}
	}
}

func TestParseKalenderkuHTML_Empty(t *testing.T) {
	holidays, err := parseKalenderkuHTML(strings.NewReader("<html><body></body></html>"), 2027)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(holidays) != 0 {
		t.Fatalf("expected 0 holidays, got %d", len(holidays))
	}
}
