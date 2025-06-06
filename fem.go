package main

import (
	"fmt"
	"log/slog"
	"math"
	"slices"
	"time"

	"gonum.org/v1/exp/linsolve"
	"gonum.org/v1/gonum/mat"
)

type ElementSide struct {
	Element int
	Side    int
}

type FEM struct {
	elements [][20][3]float64 // Coords of grid vertices in local space, npq * 20 * 3 (x, y, z)
	akt      [][3]float64     // Coords of grid vertices in global space, npq * 3 (x, y, z)
	nt       [][20]int        // Local element indexes, npq * 20

	zu map[ElementSide]bool // Fixed points, index of the element and side
	zp map[ElementSide]bool // Pushed points, index of the element and side

	dj    [][27][3][3]float64 // Jacobian matrix, npq * 27 * 3 (a, b, g) * 3 (x, y, z)
	djDet [][27]float64       // Jacobian determinant, npq * 27

	dfixyz [][27][20][3]float64 // Derivative of approximation function in global space, npq * 27 * 20 * 3 (x, y, z)

	mge [][60][60]float64 // Stiffness matrix for elements, npq * 60 * 60
	mg  [][]float64       // Stiffness matrix, npq * 3 (x, y, z) * npq * 3 (x, y, z)

	fe [][60]float64 // Forces for elements, npq * 60
	f  []float64     // Forces, npq * 3 (x, y, z)

	u []float64 // Displacements, npq * 3 (x, y, z)
}

func (f *FEM) BuildElements(bodySize [3]float64, bodySplit [3]int) ([][3]float64, map[[3]int]int) {
	stepA := bodySize[0] / float64(bodySplit[0])
	stepB := bodySize[1] / float64(bodySplit[1])
	stepC := bodySize[2] / float64(bodySplit[2])

	f.elements = nil
	for k := range bodySplit[2] {
		for j := range bodySplit[1] {
			for i := range bodySplit[0] {
				f.elements = append(f.elements, f.createCube(
					float64(i)*stepA, float64(i+1)*stepA,
					float64(j)*stepB, float64(j+1)*stepB,
					float64(k)*stepC, float64(k+1)*stepC,
				))
			}
		}
	}

	f.akt = nil
	const showInternal = false
	indexMapping := make(map[[3]int]int)
	for k := range 2*bodySplit[2] + 1 {
		if k%2 == 0 {
			for j := range 2*bodySplit[1] + 1 {
				if j%2 == 0 {
					for i := range 2*bodySplit[0] + 1 {
						if showInternal || i == 0 || j == 0 || k == 0 || i == 2*bodySplit[0] || j == 2*bodySplit[1] || k == 2*bodySplit[2] {
							indexMapping[[3]int{i, j, k}] = len(f.akt)
						}
						f.akt = append(f.akt, [3]float64{float64(i) * stepA / 2, float64(j) * stepB / 2, float64(k) * stepC / 2})
					}
				} else {
					for i := range bodySplit[0] + 1 {
						if showInternal || i == 0 || j == 0 || k == 0 || i == bodySplit[0] || j == 2*bodySplit[1] || k == 2*bodySplit[2] {
							indexMapping[[3]int{i * 2, j, k}] = len(f.akt)
						}
						f.akt = append(f.akt, [3]float64{float64(i) * stepA, float64(j) * stepB / 2, float64(k) * stepC / 2})
					}
				}
			}
		} else {
			for j := range bodySplit[1] + 1 {
				for i := range bodySplit[0] + 1 {
					if showInternal || i == 0 || j == 0 || k == 0 || i == bodySplit[0] || j == bodySplit[1] || k == 2*bodySplit[2] {
						indexMapping[[3]int{i * 2, j * 2, k}] = len(f.akt)
					}
					f.akt = append(f.akt, [3]float64{float64(i) * stepA, float64(j) * stepB, float64(k) * stepC / 2})
				}
			}
		}
	}

	f.nt = nil
	for _, cube := range f.elements {
		var ntCube [20]int
		for i, p1 := range cube {
			found := false
			for j, p2 := range f.akt {
				const eps = 1e-6
				if math.Abs(p1[0]-p2[0]) < eps && math.Abs(p1[1]-p2[1]) < eps && math.Abs(p1[2]-p2[2]) < eps {
					ntCube[i] = j
					found = true
					break
				}
			}
			if !found {
				panic("not found NT index")
			}
		}
		f.nt = append(f.nt, ntCube)
	}

	clear(f.zu)
	clear(f.zp)
	return f.akt, indexMapping
}

func (f *FEM) ApplyForce(e, nu, p float64) [][3]float64 {
	start := time.Now()
	defer func() { slog.Info("FEM", "total-time", time.Since(start)) }()

	f.dj = nil
	for _, cube := range f.elements {
		f.dj = append(f.dj, f.createDJ(cube))
	}

	f.djDet = nil
	for _, dj := range f.dj {
		var ds [27]float64
		for i, d := range dj {
			ds[i] = d[0][0]*d[1][1]*d[2][2] +
				d[0][1]*d[1][2]*d[2][0] +
				d[0][2]*d[1][0]*d[2][1] -
				d[0][2]*d[1][1]*d[2][0] - d[0][0]*
				d[1][2]*d[2][1] -
				d[0][1]*d[1][0]*d[2][2]
		}
		f.djDet = append(f.djDet, ds)
	}

	f.dfixyz = nil
	for _, dj := range f.dj {
		f.dfixyz = append(f.dfixyz, f.createDFIXYZ(dj))
	}

	l := e / ((1 + nu) * (1 - 2*nu))
	mu := e / (2 * (1 + nu))

	f.mge = nil
	for i := range f.elements {
		f.mge = append(f.mge, f.createMGE(f.dfixyz[i], f.djDet[i], l, nu, mu))
	}
	f.mg = f.calculateMG()

	f.fe = make([][60]float64, len(f.nt))
	for es, push := range f.zp {
		if push {
			for i, fe := range f.calculateFE(p, es.Side, f.choseCubeSide(f.elements[es.Element], es.Side)) {
				f.fe[es.Element][i] += fe
			}
		}
	}
	f.f = f.calculateF()

	flatMG := make([]float64, 0, len(f.mg)*len(f.mg[0]))
	for i := range f.mg {
		flatMG = append(flatMG, f.mg[i]...)
	}
	a := mat.NewDense(len(f.mg), len(f.mg[0]), flatMG)
	b := mat.NewVecDense(len(f.f), f.f)

	uVec, err := linsolve.Iterative(&matrix{Dense: a}, b, &linsolve.CG{}, nil)
	if err != nil {
		panic(err)
	}
	f.u = uVec.X.RawVector().Data

	dAKT := slices.Clone(f.akt)
	for i, u := range f.u {
		j := i / 3
		if (i+1)%3 == 1 {
			dAKT[j][0] = f.akt[j][0] + u
		} else if (i+1)%3 == 2 {
			dAKT[j][1] = f.akt[j][1] + u
		} else {
			dAKT[j][2] = f.akt[j][2] + u
		}
	}

	return dAKT
}

func (f *FEM) createCube(aStart, aEnd, bStart, bEnd, cStart, cEnd float64) [20][3]float64 {
	aSize := aEnd - aStart
	bSize := bEnd - bStart
	cSize := cEnd - cStart

	x := [20]float64{aStart, aEnd, aEnd, aStart, aStart, aEnd, aEnd, aStart,
		aStart + aSize/2, aEnd, aStart + aSize/2, aStart,
		aStart, aEnd, aEnd, aStart, aStart + aSize/2, aEnd,
		aStart + aSize/2, aStart}

	y := [20]float64{bStart, bStart, bEnd, bEnd, bStart, bStart, bEnd, bEnd,
		bStart, bStart + bSize/2, bEnd, bStart + bSize/2,
		bStart, bStart, bEnd, bEnd, bStart, bStart + bSize/2,
		bEnd, bStart + bSize/2}

	z := [20]float64{cStart, cStart, cStart, cStart, cEnd, cEnd, cEnd, cEnd,
		cStart, cStart, cStart, cStart, cStart + cSize/2,
		cStart + cSize/2, cStart + cSize/2, cStart + cSize/2,
		cEnd, cEnd, cEnd, cEnd}

	var cube [20][3]float64
	for i := range 20 {
		cube[i] = [3]float64{x[i], y[i], z[i]}
	}

	return cube
}

func (f *FEM) createDJ(cube [20][3]float64) [27][3][3]float64 {
	const eps = 1e-10

	var dj [27][3][3]float64
	for i := range 3 * 3 * 3 {
		var sumXA, sumXB, sumXG float64
		var sumYA, sumYB, sumYG float64
		var sumZA, sumZB, sumZG float64

		for j, point := range cube {
			sumXA += point[0] * dfiabg[i][j][0]
			sumXB += point[0] * dfiabg[i][j][1]
			sumXG += point[0] * dfiabg[i][j][2]

			sumYA += point[1] * dfiabg[i][j][0]
			sumYB += point[1] * dfiabg[i][j][1]
			sumYG += point[1] * dfiabg[i][j][2]

			sumZA += point[2] * dfiabg[i][j][0]
			sumZB += point[2] * dfiabg[i][j][1]
			sumZG += point[2] * dfiabg[i][j][2]
		}

		dj[i] = [3][3]float64{
			{sumXA, sumYA, sumZA},
			{sumXB, sumYB, sumZB},
			{sumXG, sumYG, sumZG},
		}

		// Added rounding for values close to -1, 0, 1
		for j := range dj[i] {
			for k := range dj[i][j] {
				v := dj[i][j][k]
				if math.Abs(v) < eps {
					v = 0
				} else if math.Abs(v-1) < eps {
					v = 1
				} else if math.Abs(v+1) < eps {
					v = -1
				}
				dj[i][j][k] = v
			}
		}
	}
	return dj
}

func (f *FEM) createDFIXYZ(dj [27][3][3]float64) [27][20][3]float64 {
	var dfixyz [27][20][3]float64
	for i, d := range dj {
		for j, points := range dfiabg[i] {
			a := mat.NewDense(3, 3, []float64{
				d[0][0], d[0][1], d[0][2],
				d[1][0], d[1][1], d[1][2],
				d[2][0], d[2][1], d[2][2],
			})
			b := mat.NewVecDense(3, points[:])

			result, err := linsolve.Iterative(&matrix{Dense: a}, b, &linsolve.GMRES{}, nil)
			if err != nil {
				panic(fmt.Errorf("compute DFIXYZ at %d %d: %w", i, j, err))
			} else {
				dfixyz[i][j] = [3]float64{result.X.AtVec(0), result.X.AtVec(1), result.X.AtVec(2)}
			}
		}
	}
	return dfixyz
}

type matrix struct {
	*mat.Dense
}

func (m *matrix) MulVecTo(dst *mat.VecDense, trans bool, x mat.Vector) {
	if trans {
		dst.MulVec(m.T(), x)
	} else {
		dst.MulVec(m.Dense, x)
	}
}

func (f *FEM) createMGE(dfixyz [27][20][3]float64, djDet [27]float64, l, nu, mu float64) [60][60]float64 {
	var matrixA11, matrixA22, matrixA33 [20][20]float64
	var matrixA12, matrixA13, matrixA23 [20][20]float64

	for i := range 20 {
		for j := range 20 {
			index := 0

			var a11, a22, a33 float64
			var a12, a13, a23 float64

			for _, m := range gaussianConst {
				for _, n := range gaussianConst {
					for _, k := range gaussianConst {
						dfi := dfixyz[index]

						a11 += m * n * k * (l*(1-nu)*(dfi[i][0]*dfi[j][0]) +
							mu*((dfi[i][1]*dfi[j][1])+(dfi[i][2]*dfi[j][2]))) * djDet[index]

						a22 += m * n * k * (l*(1-nu)*(dfi[i][1]*dfi[j][1]) +
							mu*((dfi[i][0]*dfi[j][0])+(dfi[i][2]*dfi[j][2]))) * djDet[index]

						a33 += m * n * k * (l*(1-nu)*(dfi[i][2]*dfi[j][2]) +
							mu*((dfi[i][0]*dfi[j][0])+(dfi[i][1]*dfi[j][1]))) * djDet[index]

						a12 += m * n * k * (l*nu*(dfi[i][0]*dfi[j][1]) +
							mu*(dfi[i][1]*dfi[j][0])) * djDet[index]

						a13 += m * n * k * (l*nu*(dfi[i][0]*dfi[j][2]) +
							mu*(dfi[i][2]*dfi[j][0])) * djDet[index]

						a23 += m * n * k * (l*nu*(dfi[i][1]*dfi[j][2]) +
							mu*(dfi[i][2]*dfi[j][1])) * djDet[index]

						index++
					}
				}
			}

			matrixA11[i][j] = a11
			matrixA22[i][j] = a22
			matrixA33[i][j] = a33
			matrixA12[i][j] = a12
			matrixA13[i][j] = a13
			matrixA23[i][j] = a23
		}
	}

	var mge [60][60]float64
	for i := 0; i < 20; i++ {
		for j := 0; j < 20; j++ {
			mge[i][j] = matrixA11[i][j]
			mge[i][20+j] = matrixA12[i][j]
			mge[i][40+j] = matrixA13[i][j]

			mge[20+i][j] = matrixA12[j][i]
			mge[20+i][20+j] = matrixA22[i][j]
			mge[20+i][40+j] = matrixA23[i][j]

			mge[40+i][j] = matrixA13[j][i]
			mge[40+i][20+j] = matrixA23[j][i]
			mge[40+i][40+j] = matrixA33[i][j]
		}
	}
	return mge
}

func (f *FEM) choseCubeSide(cube [20][3]float64, n int) [8][3]float64 {
	sideOfAxis := n / 2
	var coordValue float64
	if n%2 == 0 {
		coordValue = math.MaxFloat64
		for _, point := range cube {
			coordValue = min(point[sideOfAxis], coordValue)
		}
	} else {
		coordValue = -math.MaxFloat64
		for _, point := range cube {
			coordValue = max(point[sideOfAxis], coordValue)
		}
	}

	i := 0
	var indexes [8]int // TODO: Remove
	var points [8][3]float64
	for j, point := range cube {
		if point[sideOfAxis] == coordValue {
			points[i] = point
			indexes[i] = j
			i++
		}
	}

	switch n {
	case 0:
		points[0], points[1] = points[1], points[0]
		points[6], points[7] = points[7], points[6]
	case 1:
		points[2], points[3] = points[3], points[2]
		points[5], points[6], points[7] = points[6], points[7], points[5]
	case 2:
		points[2], points[3] = points[3], points[2]
		points[5], points[6], points[7] = points[6], points[7], points[5]
	case 3:
		points[2], points[3] = points[3], points[2]
		points[5], points[6], points[7] = points[6], points[7], points[5]
	case 4:
		points[0], points[1], points[2], points[3] = points[3], points[2], points[1], points[0]
		points[4], points[6] = points[6], points[4]
	case 5:
		// OK
	}

	return points
}

func (f *FEM) calculateFE(p float64, side int, zp [8][3]float64) [60]float64 {
	dXYZdNT := f.dXYZdNT(zp)
	var fe1, fe2, fe3 [8]float64

	for i := range 8 {
		index := 0
		for _, m := range gaussianConst {
			for _, n := range gaussianConst {
				dXYZdNTi := dXYZdNT[index]
				dpsiteXYZdeNTi := dpsiteXYZdeNT[index][i]
				fe1[i] += m * n * p * (dXYZdNTi[1][0]*dXYZdNTi[2][1] - dXYZdNTi[2][0]*dXYZdNTi[1][1]) * dpsiteXYZdeNTi
				fe2[i] += m * n * p * (dXYZdNTi[2][0]*dXYZdNTi[0][1] - dXYZdNTi[0][0]*dXYZdNTi[2][1]) * dpsiteXYZdeNTi
				fe3[i] += m * n * p * (dXYZdNTi[0][0]*dXYZdNTi[1][1] - dXYZdNTi[1][0]*dXYZdNTi[0][1]) * dpsiteXYZdeNTi
				index++
			}
		}
	}

	switch side {
	case 0:
		return [60]float64{
			3:  fe1[0],
			0:  fe1[1],
			4:  fe1[2],
			7:  fe1[3],
			11: fe1[4],
			12: fe1[5],
			19: fe1[6],
			15: fe1[7],
		}
	case 1:
		return [60]float64{
			1:  fe1[0],
			2:  fe1[1],
			6:  fe1[2],
			5:  fe1[3],
			9:  fe1[4],
			14: fe1[5],
			17: fe1[6],
			13: fe1[7],
		}
	case 2:
		return [60]float64{
			20 + 0:  fe2[0],
			20 + 1:  fe2[1],
			20 + 5:  fe2[2],
			20 + 4:  fe2[3],
			20 + 8:  fe2[4],
			20 + 13: fe2[5],
			20 + 16: fe2[6],
			20 + 12: fe2[7],
		}
	case 3:
		return [60]float64{
			20 + 2:  fe2[0],
			20 + 3:  fe2[1],
			20 + 7:  fe2[2],
			20 + 6:  fe2[3],
			20 + 10: fe2[4],
			20 + 15: fe2[5],
			20 + 18: fe2[6],
			20 + 14: fe2[7],
		}
	case 4:
		return [60]float64{
			40 + 3:  fe3[0],
			40 + 2:  fe3[1],
			40 + 1:  fe3[2],
			40 + 0:  fe3[3],
			40 + 10: fe3[4],
			40 + 9:  fe3[5],
			40 + 8:  fe3[6],
			40 + 11: fe3[7],
		}
	case 5:
		return [60]float64{
			40 + 4:  fe3[0],
			40 + 5:  fe3[1],
			40 + 6:  fe3[2],
			40 + 7:  fe3[3],
			40 + 16: fe3[4],
			40 + 17: fe3[5],
			40 + 18: fe3[6],
			40 + 19: fe3[7],
		}
	default:
		panic("invalid side")
	}
}

func (f *FEM) dXYZdNT(points [8][3]float64) [3 * 3][3][2]float64 {
	var dXYZdNT [9][3][2]float64
	for i := range 3 * 3 {
		var sumXEta, sumYEta, sumZEta float64
		var sumXTau, sumYTau, sumZTau float64

		for j, point := range points {
			sumXEta += point[0] * dpsite[i][j][0]
			sumYEta += point[1] * dpsite[i][j][0]
			sumZEta += point[2] * dpsite[i][j][0]
			sumXTau += point[0] * dpsite[i][j][1]
			sumYTau += point[1] * dpsite[i][j][1]
			sumZTau += point[2] * dpsite[i][j][1]
		}

		dXYZdNT[i] = [3][2]float64{
			{sumXEta, sumXTau},
			{sumYEta, sumYTau},
			{sumZEta, sumZTau},
		}
	}
	return dXYZdNT
}

func (f *FEM) calculateMG() [][]float64 {
	mg := make([][]float64, 3*len(f.akt))
	for i := 0; i < len(mg); i++ {
		mg[i] = make([]float64, 3*len(f.akt))
	}

	for k, mge := range f.mge {
		for j := range 60 {
			for i := range 60 {
				var iForNT, xyzCoordI int
				if i < 20 {
					iForNT = i
					xyzCoordI = 0
				} else if i < 40 {
					iForNT = i - 20
					xyzCoordI = 1
				} else {
					iForNT = i - 40
					xyzCoordI = 2
				}

				var jForNT, xyzCoordJ int
				if j < 20 {
					jForNT = j
					xyzCoordJ = 0
				} else if j < 40 {
					jForNT = j - 20
					xyzCoordJ = 1
				} else {
					jForNT = j - 40
					xyzCoordJ = 2
				}

				mgI := 3*f.nt[k][iForNT] + xyzCoordI
				mgJ := 3*f.nt[k][jForNT] + xyzCoordJ
				mg[mgJ][mgI] += mge[j][i]
			}
		}
	}

	for i, point := range f.akt {
		keep := false
		for es, fix := range f.zu {
			if !fix {
				continue
			}

			points := f.choseCubeSide(f.elements[es.Element], es.Side)
			if slices.Contains(points[:], point) {
				keep = true
			}
		}
		if !keep {
			continue
		}

		ix := 3*i + 0
		iy := 3*i + 1
		iz := 3*i + 2
		mg[ix][ix] = 1e16
		mg[iy][iy] = 1e16
		mg[iz][iz] = 1e16
	}

	return mg
}

func (f *FEM) calculateF() []float64 {
	fr := make([]float64, 3*len(f.akt))

	for j, fe := range f.fe {
		for i := range 60 {
			var iForNT, xyzCoordI int
			if i < 20 {
				iForNT = i
				xyzCoordI = 0
			} else if i < 40 {
				iForNT = i - 20
				xyzCoordI = 1
			} else {
				iForNT = i - 40
				xyzCoordI = 2
			}

			fI := 3*f.nt[j][iForNT] + xyzCoordI
			fr[fI] += fe[i]
		}
	}

	return fr
}
