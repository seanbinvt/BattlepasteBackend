package battlereportparsing

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

//PlayerInfo - Information for a player within a battle report
type PlayerInfo struct {
	PlayerTag   string  `bson:"PlayerTag"`
	PlayerName  string  `bson:"PlayerName"`
	PlayerLevel float64 `bson:"PlayerLevel"`
	FleetName   string  `bson:"FleetName"`
	OnBase      bool    `bson:"OnBase"`
}

//Unit - Resultant information for a single unit from a battle report
type Unit struct {
	Name       string  `bson:"Name"`
	StartQuant float64 `bson:"StartQuant"`
	EndQuant   float64 `bson:"EndQuant"`
	Power      float64 `bson:"Power"`
	Armour     float64 `bson:"Armour"`
	Shield     float64 `bson:"Shield"`
}

//BattleReport - Required information for a successful battle report
type BattleReport struct {
	BaseName   string `bson:"BaseName"`
	Coordinate string `bson:"Coordinate"`
	Date       string `bson:"Date"`
	Time       string `bson:"Time"`
	Server     string `bson:"Server"`

	Attacker          PlayerInfo `bson:"Attacker"`
	AttackerUnits     []Unit     `bson:"AttackerUnits"`
	AttackerDestroyed bool       `bson:"AttackerDestroyed"`
	Defender          PlayerInfo `bson:"Defender"`
	DefenderUnits     []Unit     `bson:"DefenderUnits"`
	DefenderDestroyed bool       `bson:"DefenderDestroyed"`

	BaseID         uint32  `bson:"BaseID"`
	StartDefenses  float64 `bson:"StartDefenses"`
	EndDefenses    float64 `bson:"EndDefenses"`
	CommandCenters uint16  `bson:"CommandCenters"`
	Commander      string  `bson:"Commander"`

	//Add something later to account for United Colonies/Drekons not showing EXP
	AttackerLoss uint32 `bson:"AttackerLoss"`
	DefenderLoss uint32 `bson:"DefenderLoss"`
	TotalLoss    uint32 `bson:"TotalLoss"`
	AttackerExp  uint32 `bson:"AttackerExp"`
	DefenderExp  uint32 `bson:"DefenderExp"`
	DebrisGen    uint32 `bson:"DebrisGen"`

	BaseAttack      bool   `bson:"BaseAttack"`      // if a base was directly attacked
	BaseConquer     bool   `bson:"BaseConquer"`     // if the attacked base was successfully occupied
	BasePillaged    bool   `bson:"BasePillaged"`    // if the attacked base was successfully pillaged
	BasePillage     uint32 `bson:"BasePillage"`     //  pillage #
	CommanderKilled bool   `bson:"CommanderKilled"` // If the commander of the base was killed

	TechDate time.Time `bson:"TechDate` // The date where the tech is unlocked for viewing
}

/*Parse
Initialized parsing to send BattleReport object back to main.

Will return a true bool if there was an error encountered during the parsing.
*/
func Parse(report string) (BattleReport, bool) {
	var finalReport BattleReport
	var commandChecked bool

	//report, _ := ioutil.ReadFile("report2.txt")
	reportAlt := strings.Split(string(report), "\n")

	lastIndex := 0 //Keep track of which line has been iterated through

	// Get LOCATION, TIME, SERVER
	//fmt.Println("getHeaderStart", " - ", lastIndex+1)
	serverCheck := getHeader(&finalReport, reportAlt, &lastIndex)
	if !serverCheck {
		fmt.Println("here")
		return finalReport, true
	}
	//fmt.Println("getHeader", " - ", lastIndex+1)

	// Get  (Attack Force) PLAYER, PLAYER NAME - Check COMMAND CENTERS,  COMMANDER
	//fmt.Println("getAttackerInformationStart", " - ", lastIndex+1)
	getAttackerInformation(&finalReport, reportAlt, &lastIndex, &commandChecked)
	//fmt.Println("getAttackerInformation", " - ", lastIndex+1)

	// Get (Defense Force) PLAYER, FLEET NAME - Check BASEID, START DEF, END DEF, COMMAND CENTERS,  COMMANDER
	//fmt.Println("getDefenderInformationStart", " - ", lastIndex+1)
	getDefenderInformation(&finalReport, reportAlt, &lastIndex, &commandChecked)
	//fmt.Println("getDefenderInformation", " - ", lastIndex+1)

	// Get ATTACK FORCE UNITS
	lastIndex += 3
	//fmt.Println("attackerUnitsStart", " - ", lastIndex+1)
	finalReport.AttackerUnits, finalReport.AttackerDestroyed = fleetParse(reportAlt, &lastIndex)
	//fmt.Println("attackerUnits", " - ", lastIndex+1)

	// Get DEFENSE FORCE UNITS
	lastIndex += 3
	//fmt.Println("defenderUnitsStart", " - ", lastIndex+1)
	finalReport.DefenderUnits, finalReport.DefenderDestroyed = fleetParse(reportAlt, &lastIndex)
	//fmt.Println("defenderUnits", " - ", lastIndex+1)

	// Get UNITS DESROYED, EXPERIENCE (if not NPC), DEBRIS CREATED, CONQUER STATUS, PILLAGE (if)\
	lastIndex++
	//fmt.Println("footerInfoStart", " - ", lastIndex+1)
	getFooterInformation(&finalReport, reportAlt, &lastIndex, &commandChecked)
	//fmt.Println("footerInfo", " - ", lastIndex+1)

	checkDestroyed(&finalReport)

	return finalReport, false
}

func checkServers(server string) bool {
	serversv1 := [...]string{"Alpha", "Beta", "Ceti", "Delta", "Epsilon", "Fenix", "Gamma", "Helion", "Ixion", "Juno", "Kappa", "Lyra", "Mira", "Nova", "Omega", "Pegasus", "Quantum", "Rigel", "Sigma", "Typhon", "Utopia"}
	serversv2 := [...]string{"Andromeda", "Bravo", "Centauri", "Drako", "Elysium", "Frontier", "Gaia", "Hydra", "Iridium", "Jade", "Kepler", "Lynx", "Mystic", "Nexus"}
	serversv3 := [...]string{"Ares", "Antares"}

	for _, s := range serversv1 {
		if server == strings.ToLower(s) {
			return true
		}
	}

	for _, s := range serversv2 {
		if server == strings.ToLower(s) {
			return true
		}
	}

	for _, s := range serversv3 {
		if server == strings.ToLower(s) {
			return true
		}
	}

	return false
}

// Check if a fleet was destroyed
func checkDestroyed(report *BattleReport) {
	if report.AttackerDestroyed {
		//fmt.Println("Attacker FleetName: ", report.Attacker.FleetName)
		name := strings.Fields(report.Attacker.FleetName)
		report.Attacker.FleetName = strings.Join(name[:len(name)-1], " ")
	}

	if report.DefenderDestroyed {
		name := strings.Fields(report.Defender.FleetName)
		report.Defender.FleetName = strings.Join(name[:len(name)-1], " ")
	}
}

func fleetParse(lines []string, lastIndex *int) ([]Unit, bool) {
	var units []Unit
	index := *lastIndex

	destroyed := true

	/* if line := strings.Fields(lines[index]); len(line) < 6 {
		index++
	} */

	for line := strings.Fields(lines[index]); len(line) >= 6; index++ {
		if line[0] != "Unit" {
			var unit Unit
			length := len(line) - 1

			unit.Name = strings.Join(line[:length-4], " ")
			unit.StartQuant, _ = strconv.ParseFloat(line[length-4], 64)
			unit.EndQuant, _ = strconv.ParseFloat(line[length-3], 64)
			unit.Power, _ = strconv.ParseFloat(line[length-2], 64)
			unit.Armour, _ = strconv.ParseFloat(line[length-1], 64)
			unit.Shield, _ = strconv.ParseFloat(line[length], 64)

			if unit.EndQuant > 0 {
				destroyed = false
			}

			units = append(units, unit)
		}
		line = strings.Fields(lines[index+1])
	}
	*lastIndex = index
	return units, destroyed
}

func baseInfoParse(lines []string, lastIndex *int, commandChecked *bool, player *PlayerInfo) (uint16, string) {
	var commandCenters int
	var commander string

	line := strings.Fields(lines[*lastIndex])
	//fmt.Println(*lastIndex, "baseInfoParse", line)
	if len(line) > 0 {
		if line[0] == "Command" {
			// Check Commander center line
			player.OnBase = true
			commandCenters, _ = strconv.Atoi(line[len(line)-1])
			*commandChecked = true
			*lastIndex++

			// Check Commander line
			line = strings.Fields(lines[*lastIndex])
			if len(line) > 0 {
				commander = strings.Join(line[1:], " ")
			} else {
				*lastIndex--
			}
		}
	} else {
		*lastIndex--
	}
	*lastIndex++
	//fmt.Println(lines[*lastIndex])

	return uint16(commandCenters), commander
}

func playerParse(lines []string, lastLine *int) (string, string, float64, string) {
	var tag, name, fleetName string
	var level float64

	lineAlt := strings.Fields(lines[*lastLine])

	// Check if player is an NPC
	if (lineAlt[1] == "United" && lineAlt[2] == "Colonies") || (lineAlt[1] == "Drekons") {
		name = strings.Join(lineAlt[1:], " ")
		*lastLine++
	} else {
		// Get player level
		level, _ = strconv.ParseFloat(lineAlt[len(lineAlt)-1], 64)
		level = math.Round(level*100) / 100

		// Check if player is guilded
		if strings.Contains(lines[*lastLine], "[") {
			tag = lineAlt[1]
			name = strings.Join(lineAlt[2:len(lineAlt)-2], " ")
		} else if !strings.Contains(lines[*lastLine], "United") {
			name = strings.Join(lineAlt[1:len(lineAlt)-2], " ")
		}
		*lastLine++
	}

	lineAlt = strings.Fields(lines[*lastLine])

	fleetName = strings.Join(lineAlt[2:], " ")
	*lastLine++

	return tag, name, level, fleetName
}

func getHeader(finalReport *BattleReport, reportAlt []string, lastIndex *int) bool {
	// GET COORDS and BASE NAME
	for x, s := range reportAlt {
		if strings.Contains(s, "Location") {
			temp := strings.Fields(s)
			replacer := strings.NewReplacer("(", "", ")", "")
			finalReport.Coordinate = replacer.Replace(temp[len(temp)-1])

			if len(temp) > 2 {
				finalReport.BaseName = strings.Join(temp[1:len(temp)-1], " ")
			}
			*lastIndex = x
			break
		}
	}
	*lastIndex++

	// GET DATE and TIME
	dateLine := strings.Fields(reportAlt[*lastIndex])
	finalReport.Date, finalReport.Time = dateLine[1], dateLine[2]
	*lastIndex++

	// GET SERVER NAME
	finalReport.Server = strings.ToLower(strings.Fields(reportAlt[*lastIndex])[1])
	*lastIndex += 2

	return checkServers(finalReport.Server)
}

func getAttackerInformation(finalReport *BattleReport, reportAlt []string, lastIndex *int, commandChecked *bool) {
	// GET ATTACKER INFORMATION
	finalReport.Attacker.PlayerTag, finalReport.Attacker.PlayerName, finalReport.Attacker.PlayerLevel, finalReport.Attacker.FleetName = playerParse(reportAlt, lastIndex)
	//fmt.Println("SDADFSAFDSAFDSAFD", *lastIndex)
	// CHECK IF ATTACKER ON BASE INFORMATION
	finalReport.CommandCenters, finalReport.Commander = baseInfoParse(reportAlt, lastIndex, commandChecked, &finalReport.Attacker)
}

func getDefenderInformation(finalReport *BattleReport, reportAlt []string, lastIndex *int, commandChecked *bool) {
	if strings.Fields(reportAlt[*lastIndex])[0] == "Defensive" {
		*lastIndex++
	}
	// GET DEFENDER INFORMATION
	finalReport.Defender.PlayerTag, finalReport.Defender.PlayerName, finalReport.Defender.PlayerLevel, finalReport.Defender.FleetName = playerParse(reportAlt, lastIndex)

	//fmt.Println(*lastIndex, "HHSHSHSHSH", *commandChecked)
	//CHECK IF DEFENDER ON BASE INFORMATION
	if !*commandChecked {
		if line := strings.Fields(reportAlt[*lastIndex]); strings.Contains(reportAlt[*lastIndex], "Base") {
			// BASE ID
			if line[0] == "Base" {
				n, _ := strconv.Atoi(line[len(line)-1])
				finalReport.BaseID = uint32(n)
				*lastIndex++

				// START DEFENSE PERCENT
				line = strings.Fields(reportAlt[*lastIndex])
				str := line[len(line)-1]
				num := line[len(line)-1][:len(str)-1]
				finalReport.StartDefenses, _ = strconv.ParseFloat(num, 64)
				*lastIndex++

				// END DEFENSE PERCENT
				line = strings.Fields(reportAlt[*lastIndex])
				str = line[len(line)-1]
				num = line[len(line)-1][:len(str)-1]
				finalReport.EndDefenses, _ = strconv.ParseFloat(num, 64)
				*lastIndex++
			}
		}
		//fmt.Println(*lastIndex, "------------------------", *commandChecked)
		finalReport.CommandCenters, finalReport.Commander = baseInfoParse(reportAlt, lastIndex, commandChecked, &finalReport.Defender)
	}
	//*lastIndex += 3
}

func getFooterInformation(finalReport *BattleReport, reportAlt []string, lastIndex *int, commandChecked *bool) {
	line := strings.Fields(reportAlt[*lastIndex])
	//fmt.Println("ppppp", line)

	/* for len(line) < 2 || line[0] != "Total" {
		fmt.Println(line)
		*lastIndex++
		line = strings.Fields(reportAlt[*lastIndex])
	} */

	// Get losses line
	n, _ := strconv.Atoi(line[5])
	finalReport.TotalLoss = uint32(n)
	n, _ = strconv.Atoi(line[8])
	finalReport.AttackerLoss = uint32(n)
	n, _ = strconv.Atoi(line[11])
	finalReport.DefenderLoss = uint32(n)
	*lastIndex++

	// Get experience line
	if finalReport.Attacker.PlayerName != "United Colonies" && finalReport.Attacker.PlayerName != "Drekons" && finalReport.Defender.PlayerName != "United Colonies" && finalReport.Defender.PlayerName != "Drekons" {
		line = strings.Fields(reportAlt[*lastIndex])
		n, _ := strconv.Atoi(line[3][1:])
		finalReport.AttackerExp = uint32(n)
		n, _ = strconv.Atoi(line[6][1:])
		finalReport.DefenderExp = uint32(n)
		*lastIndex++
	}

	// Get debris line
	line = strings.Fields(reportAlt[*lastIndex])
	n, _ = strconv.Atoi(line[4])
	finalReport.DebrisGen = uint32(n)
	*lastIndex++

	//fmt.Println(len(reportAlt)-1, " vs ", *lastIndex)

	if *lastIndex <= len(reportAlt)-1 {
		line = strings.Fields(reportAlt[*lastIndex])
		if line[1] == "conquered" {
			finalReport.BaseConquer = true
			*lastIndex++

			if *lastIndex <= len(reportAlt)-1 {
				line = strings.Fields(reportAlt[*lastIndex])
				if line[0] == "Commander" {
					finalReport.CommanderKilled = true
					*lastIndex++
				}
			}

			if *lastIndex <= len(reportAlt)-1 {
				line = strings.Fields(reportAlt[*lastIndex])
				n, _ = strconv.Atoi(line[2])
				finalReport.BasePillage = uint32(n)
				finalReport.BasePillaged = true
			}
		}
		finalReport.BaseAttack = true
	}
}
