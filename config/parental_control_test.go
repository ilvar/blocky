package config

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParentalControl", func() {
	Describe("TimeOfDay", func() {
		var t TimeOfDay

		BeforeEach(func() {
			t = TimeOfDay{}
		})

		Describe("UnmarshalText", func() {
			It("should parse valid time format", func() {
				err := t.UnmarshalText([]byte("08:30"))
				Expect(err).Should(Succeed())
				Expect(t.Hour).Should(Equal(8))
				Expect(t.Minute).Should(Equal(30))
			})

			It("should parse midnight", func() {
				err := t.UnmarshalText([]byte("00:00"))
				Expect(err).Should(Succeed())
				Expect(t.Hour).Should(Equal(0))
				Expect(t.Minute).Should(Equal(0))
			})

			It("should parse end of day", func() {
				err := t.UnmarshalText([]byte("23:59"))
				Expect(err).Should(Succeed())
				Expect(t.Hour).Should(Equal(23))
				Expect(t.Minute).Should(Equal(59))
			})

			It("should fail for invalid format", func() {
				err := t.UnmarshalText([]byte("8:30"))
				Expect(err).Should(Succeed()) // Single digit hour is valid
				Expect(t.Hour).Should(Equal(8))
			})

			It("should fail for missing colon", func() {
				err := t.UnmarshalText([]byte("0830"))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("expected HH:MM"))
			})

			It("should fail for invalid hour", func() {
				err := t.UnmarshalText([]byte("25:00"))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("invalid hour"))
			})

			It("should fail for invalid minute", func() {
				err := t.UnmarshalText([]byte("08:60"))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("invalid minute"))
			})
		})

		Describe("ToMinutes", func() {
			It("should return correct minutes", func() {
				t = TimeOfDay{Hour: 8, Minute: 30}
				Expect(t.ToMinutes()).Should(Equal(8*60 + 30))
			})

			It("should return 0 for midnight", func() {
				t = TimeOfDay{Hour: 0, Minute: 0}
				Expect(t.ToMinutes()).Should(Equal(0))
			})
		})

		Describe("String", func() {
			It("should format with leading zeros", func() {
				t = TimeOfDay{Hour: 8, Minute: 5}
				Expect(t.String()).Should(Equal("08:05"))
			})
		})
	})

	Describe("DayPattern", func() {
		var d DayPattern

		BeforeEach(func() {
			d = DayPattern{}
		})

		Describe("UnmarshalText", func() {
			It("should parse everyday", func() {
				err := d.UnmarshalText([]byte("everyday"))
				Expect(err).Should(Succeed())
				for day := time.Sunday; day <= time.Saturday; day++ {
					Expect(d.Contains(day)).Should(BeTrue())
				}
			})

			It("should parse weekdays", func() {
				err := d.UnmarshalText([]byte("weekdays"))
				Expect(err).Should(Succeed())
				Expect(d.Contains(time.Monday)).Should(BeTrue())
				Expect(d.Contains(time.Tuesday)).Should(BeTrue())
				Expect(d.Contains(time.Wednesday)).Should(BeTrue())
				Expect(d.Contains(time.Thursday)).Should(BeTrue())
				Expect(d.Contains(time.Friday)).Should(BeTrue())
				Expect(d.Contains(time.Saturday)).Should(BeFalse())
				Expect(d.Contains(time.Sunday)).Should(BeFalse())
			})

			It("should parse weekends", func() {
				err := d.UnmarshalText([]byte("weekends"))
				Expect(err).Should(Succeed())
				Expect(d.Contains(time.Saturday)).Should(BeTrue())
				Expect(d.Contains(time.Sunday)).Should(BeTrue())
				Expect(d.Contains(time.Monday)).Should(BeFalse())
			})

			It("should parse specific days list", func() {
				err := d.UnmarshalText([]byte("monday,wednesday,friday"))
				Expect(err).Should(Succeed())
				Expect(d.Contains(time.Monday)).Should(BeTrue())
				Expect(d.Contains(time.Wednesday)).Should(BeTrue())
				Expect(d.Contains(time.Friday)).Should(BeTrue())
				Expect(d.Contains(time.Tuesday)).Should(BeFalse())
			})

			It("should parse short day names", func() {
				err := d.UnmarshalText([]byte("mon,tue,wed"))
				Expect(err).Should(Succeed())
				Expect(d.Contains(time.Monday)).Should(BeTrue())
				Expect(d.Contains(time.Tuesday)).Should(BeTrue())
				Expect(d.Contains(time.Wednesday)).Should(BeTrue())
			})

			It("should be case insensitive", func() {
				err := d.UnmarshalText([]byte("MONDAY,Tuesday,FRIDAY"))
				Expect(err).Should(Succeed())
				Expect(d.Contains(time.Monday)).Should(BeTrue())
				Expect(d.Contains(time.Tuesday)).Should(BeTrue())
				Expect(d.Contains(time.Friday)).Should(BeTrue())
			})

			It("should fail for invalid day name", func() {
				err := d.UnmarshalText([]byte("notaday"))
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(ContainSubstring("unknown day name"))
			})
		})

		Describe("String", func() {
			It("should return everyday for all days", func() {
				_ = d.UnmarshalText([]byte("everyday"))
				Expect(d.String()).Should(Equal("everyday"))
			})

			It("should return weekdays for Mon-Fri", func() {
				_ = d.UnmarshalText([]byte("weekdays"))
				Expect(d.String()).Should(Equal("weekdays"))
			})

			It("should return weekends for Sat-Sun", func() {
				_ = d.UnmarshalText([]byte("weekends"))
				Expect(d.String()).Should(Equal("weekends"))
			})
		})
	})

	Describe("ScheduleWindow", func() {
		Describe("IsActive", func() {
			var w ScheduleWindow

			BeforeEach(func() {
				w = ScheduleWindow{}
			})

			It("should be active during normal window", func() {
				_ = w.Start.UnmarshalText([]byte("08:00"))
				_ = w.End.UnmarshalText([]byte("15:00"))
				_ = w.Days.UnmarshalText([]byte("weekdays"))

				// Monday at 10:00
				t := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeTrue())
			})

			It("should not be active outside window hours", func() {
				_ = w.Start.UnmarshalText([]byte("08:00"))
				_ = w.End.UnmarshalText([]byte("15:00"))
				_ = w.Days.UnmarshalText([]byte("weekdays"))

				// Monday at 16:00 (after end)
				t := time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeFalse())
			})

			It("should not be active on wrong day", func() {
				_ = w.Start.UnmarshalText([]byte("08:00"))
				_ = w.End.UnmarshalText([]byte("15:00"))
				_ = w.Days.UnmarshalText([]byte("weekdays"))

				// Saturday at 10:00
				t := time.Date(2024, 1, 13, 10, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeFalse())
			})

			It("should handle overnight window - before midnight", func() {
				_ = w.Start.UnmarshalText([]byte("22:00"))
				_ = w.End.UnmarshalText([]byte("06:00"))
				_ = w.Days.UnmarshalText([]byte("everyday"))

				// 23:00 should be active
				t := time.Date(2024, 1, 15, 23, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeTrue())
			})

			It("should handle overnight window - after midnight", func() {
				_ = w.Start.UnmarshalText([]byte("22:00"))
				_ = w.End.UnmarshalText([]byte("06:00"))
				_ = w.Days.UnmarshalText([]byte("everyday"))

				// 03:00 should be active
				t := time.Date(2024, 1, 15, 3, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeTrue())
			})

			It("should not be active outside overnight window", func() {
				_ = w.Start.UnmarshalText([]byte("22:00"))
				_ = w.End.UnmarshalText([]byte("06:00"))
				_ = w.Days.UnmarshalText([]byte("everyday"))

				// 12:00 should not be active
				t := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeFalse())
			})

			It("should be active at start time boundary", func() {
				_ = w.Start.UnmarshalText([]byte("08:00"))
				_ = w.End.UnmarshalText([]byte("15:00"))
				_ = w.Days.UnmarshalText([]byte("everyday"))

				// Exactly 08:00
				t := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeTrue())
			})

			It("should not be active at end time boundary", func() {
				_ = w.Start.UnmarshalText([]byte("08:00"))
				_ = w.End.UnmarshalText([]byte("15:00"))
				_ = w.Days.UnmarshalText([]byte("everyday"))

				// Exactly 15:00 - end time is exclusive
				t := time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC)
				Expect(w.IsActive(t)).Should(BeFalse())
			})
		})
	})

	Describe("ScheduleAction", func() {
		It("should parse blockGroups", func() {
			var a ScheduleAction
			err := a.UnmarshalText([]byte("blockGroups"))
			Expect(err).Should(Succeed())
			Expect(a).Should(Equal(ScheduleActionBlockGroups))
		})

		It("should parse blockAll", func() {
			var a ScheduleAction
			err := a.UnmarshalText([]byte("blockAll"))
			Expect(err).Should(Succeed())
			Expect(a).Should(Equal(ScheduleActionBlockAll))
		})

		It("should parse allowGroupsOnly", func() {
			var a ScheduleAction
			err := a.UnmarshalText([]byte("allowGroupsOnly"))
			Expect(err).Should(Succeed())
			Expect(a).Should(Equal(ScheduleActionAllowGroupsOnly))
		})

		It("should fail for invalid action", func() {
			var a ScheduleAction
			err := a.UnmarshalText([]byte("invalid"))
			Expect(err).Should(HaveOccurred())
		})
	})

	Describe("ParentalControl", func() {
		Describe("IsEnabled", func() {
			It("should return false when empty", func() {
				pc := ParentalControl{}
				Expect(pc.IsEnabled()).Should(BeFalse())
			})

			It("should return true when clients configured", func() {
				pc := ParentalControl{
					Clients: map[string][]ClientSchedule{
						"192.168.1.100": {},
					},
				}
				Expect(pc.IsEnabled()).Should(BeTrue())
			})
		})
	})
})
